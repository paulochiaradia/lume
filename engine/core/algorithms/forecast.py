import pandas as pd
import numpy as np
import logging

log = logging.getLogger(__name__)


def calcular_forecast(
    df_itens: pd.DataFrame,
    dias_futuro: int = 90,
    produto_key: str = None,
) -> pd.DataFrame:
    """
    Calcula forecast de demanda usando Prophet ou fallback para
    média móvel quando não há dados suficientes.

    Args:
        df_itens: DataFrame com colunas produto_key, quantidade, data_venda
        dias_futuro: Quantos dias à frente projetar
        produto_key: Se informado, projeta apenas esse produto

    Returns:
        DataFrame com previsão de demanda por produto por dia
    """
    if df_itens.empty:
        log.warning("Forecast: DataFrame vazio")
        return pd.DataFrame()

    df = df_itens.copy()
    df["data_venda"] = pd.to_datetime(df["data_venda"], utc=True)
    df["data"]       = df["data_venda"].dt.date

    # Filtra produto específico se informado
    if produto_key:
        df = df[df["produto_key"] == produto_key]

    produtos = df["produto_key"].unique()
    resultados = []

    for prod in produtos:
        df_prod = df[df["produto_key"] == prod].copy()

        # Agrega quantidade diária
        diario = df_prod.groupby("data")["quantidade"].sum().reset_index()
        diario.columns = ["ds", "y"]
        diario["ds"] = pd.to_datetime(diario["ds"])

        if len(diario) < 30:
            # Poucos dados — usa média móvel simples
            resultado = _forecast_media_movel(diario, prod, dias_futuro)
        else:
            # Dados suficientes — tenta Prophet
            resultado = _forecast_prophet(diario, prod, dias_futuro)

        if resultado is not None:
            resultados.append(resultado)

    if not resultados:
        return pd.DataFrame()

    df_final = pd.concat(resultados, ignore_index=True)
    log.info(f"Forecast calculado para {len(produtos)} produtos")
    return df_final


def _forecast_prophet(
    diario: pd.DataFrame,
    produto_key: str,
    dias_futuro: int,
) -> pd.DataFrame:
    """Forecast usando Facebook Prophet"""
    try:
        from prophet import Prophet
        import warnings
        warnings.filterwarnings("ignore")

        model = Prophet(
            yearly_seasonality=True,
            weekly_seasonality=True,
            daily_seasonality=False,
            seasonality_mode="multiplicative",
            interval_width=0.95,
        )

        model.fit(diario)

        future = model.make_future_dataframe(periods=dias_futuro)
        forecast = model.predict(future)

        # Pega apenas os dias futuros
        ultimo_dia = diario["ds"].max()
        forecast_futuro = forecast[forecast["ds"] > ultimo_dia][[
            "ds", "yhat", "yhat_lower", "yhat_upper"
        ]].copy()

        # Garante valores positivos
        forecast_futuro["yhat"]       = forecast_futuro["yhat"].clip(lower=0).round(2)
        forecast_futuro["yhat_lower"] = forecast_futuro["yhat_lower"].clip(lower=0).round(2)
        forecast_futuro["yhat_upper"] = forecast_futuro["yhat_upper"].clip(lower=0).round(2)

        forecast_futuro["produto_key"] = produto_key
        forecast_futuro["metodo"]      = "prophet"

        return forecast_futuro.rename(columns={
            "ds":          "data",
            "yhat":        "quantidade_prevista",
            "yhat_lower":  "quantidade_min",
            "yhat_upper":  "quantidade_max",
        })

    except Exception as e:
        log.warning(f"Prophet falhou para {produto_key}: {e} — usando média móvel")
        return _forecast_media_movel(diario, produto_key, dias_futuro)


def _forecast_media_movel(
    diario: pd.DataFrame,
    produto_key: str,
    dias_futuro: int,
) -> pd.DataFrame:
    """Forecast simples usando média móvel dos últimos 30 dias"""
    if diario.empty:
        return None

    media  = diario["y"].tail(30).mean()
    desvio = diario["y"].tail(30).std()

    if pd.isna(desvio):
        desvio = media * 0.2

    ultimo_dia = diario["ds"].max()
    datas_futuras = pd.date_range(
        start=ultimo_dia + pd.Timedelta(days=1),
        periods=dias_futuro
    )

    resultado = pd.DataFrame({
        "data":                datas_futuras,
        "quantidade_prevista": round(media, 2),
        "quantidade_min":      round(max(0, media - desvio), 2),
        "quantidade_max":      round(media + desvio, 2),
        "produto_key":         produto_key,
        "metodo":              "media_movel",
    })

    return resultado

def get_forecast_resumo(df_forecast: pd.DataFrame, dias: int = 30) -> pd.DataFrame:
    """
    Retorna resumo do forecast para os próximos N dias por produto.
    """
    if df_forecast.empty:
        return pd.DataFrame()

    df = df_forecast.copy()
    df["data"] = pd.to_datetime(df["data"], utc=True)

    # Usa a primeira data do forecast como referência
    # em vez de hoje — compatível com dados históricos
    data_inicio = df["data"].min()
    data_fim    = data_inicio + pd.Timedelta(days=dias)

    futuro = df[(df["data"] >= data_inicio) & (df["data"] <= data_fim)]

    resumo = futuro.groupby("produto_key").agg(
        quantidade_prevista = ("quantidade_prevista", "sum"),
        quantidade_min      = ("quantidade_min", "sum"),
        quantidade_max      = ("quantidade_max", "sum"),
        metodo              = ("metodo", "first"),
    ).round(2).reset_index()

    return resumo.sort_values("quantidade_prevista", ascending=False)