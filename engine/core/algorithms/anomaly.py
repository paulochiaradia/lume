import pandas as pd
import numpy as np
import logging

log = logging.getLogger(__name__)


def detectar_anomalias_vendas(df_vendas: pd.DataFrame) -> pd.DataFrame:
    """
    Detecta anomalias em vendas usando Isolation Forest.
    Identifica vendas com valores, descontos ou padrões fora do normal.

    Args:
        df_vendas: DataFrame com colunas total, desconto, data_venda

    Returns:
        DataFrame com flag de anomalia e score
    """
    if df_vendas.empty or len(df_vendas) < 50:
        log.warning("Anomalia: dados insuficientes")
        return pd.DataFrame()

    try:
        from sklearn.ensemble import IsolationForest
        from sklearn.preprocessing import StandardScaler
    except ImportError:
        log.error("scikit-learn não instalado")
        return pd.DataFrame()

    df = df_vendas.copy()
    df["data_venda"] = pd.to_datetime(df["data_venda"], utc=True)
    df["hora"]       = df["data_venda"].dt.hour
    df["dia_semana"] = df["data_venda"].dt.dayofweek

    # Features para o modelo
    features = ["total", "desconto", "hora", "dia_semana"]
    df["desconto"] = df["desconto"].fillna(0)

    X = df[features].fillna(0)

    # Normaliza as features
    scaler = StandardScaler()
    X_scaled = scaler.fit_transform(X)

    # Isolation Forest
    model = IsolationForest(
        contamination=0.05,  # assume 5% de anomalias
        random_state=42,
        n_estimators=100,
    )

    df["anomalia_flag"]  = model.fit_predict(X_scaled)
    df["anomalia_score"] = model.score_samples(X_scaled)

    # -1 = anomalia, 1 = normal
    df["is_anomalia"] = df["anomalia_flag"] == -1

    anomalias = df[df["is_anomalia"]].copy()
    anomalias = anomalias.sort_values("anomalia_score")

    log.info(
        f"Anomalias detectadas: {len(anomalias)} de {len(df)} vendas "
        f"({len(anomalias)/len(df)*100:.1f}%)"
    )

    return anomalias[[
        "venda_key", "data_venda", "total",
        "desconto", "hora", "anomalia_score"
    ]].reset_index(drop=True)


def detectar_anomalias_desconto(df_vendas: pd.DataFrame) -> pd.DataFrame:
    """
    Detecta vendas com desconto anormalmente alto.
    Usa Z-score para identificar outliers.
    """
    if df_vendas.empty:
        return pd.DataFrame()

    df = df_vendas.copy()
    df["desconto"] = df["desconto"].fillna(0)
    df["pct_desconto"] = df["desconto"] / df["total"].replace(0, np.nan) * 100
    df["pct_desconto"] = df["pct_desconto"].fillna(0)

    # Z-score
    media  = df["pct_desconto"].mean()
    desvio = df["pct_desconto"].std()

    if desvio == 0:
        return pd.DataFrame()

    df["z_score"] = (df["pct_desconto"] - media) / desvio

    # Considera anomalia se Z-score > 2 (2 desvios padrão acima da média)
    anomalias = df[df["z_score"] > 2].copy()
    anomalias = anomalias.sort_values("z_score", ascending=False)

    log.info(f"Descontos anômalos: {len(anomalias)}")

    return anomalias[[
        "venda_key", "data_venda", "vendedor_id",
        "total", "desconto", "pct_desconto", "z_score"
    ]].reset_index(drop=True)


def detectar_anomalias_por_vendedor(df_vendas: pd.DataFrame) -> pd.DataFrame:
    """
    Analisa padrões por vendedor e detecta desvios.
    Identifica vendedores com ticket médio ou desconto fora do padrão.
    """
    if df_vendas.empty:
        return pd.DataFrame()

    df = df_vendas.copy()
    df["desconto"] = df["desconto"].fillna(0)
    df["pct_desconto"] = df["desconto"] / df["total"].replace(0, np.nan) * 100
    df["pct_desconto"] = df["pct_desconto"].fillna(0)

    resumo = df.groupby("vendedor_id").agg(
        total_vendas    = ("venda_key", "count"),
        ticket_medio    = ("total", "mean"),
        desconto_medio  = ("pct_desconto", "mean"),
        total_faturado  = ("total", "sum"),
    ).round(2).reset_index()

    # Z-score do ticket médio entre vendedores
    if len(resumo) > 1:
        media_ticket  = resumo["ticket_medio"].mean()
        desvio_ticket = resumo["ticket_medio"].std()

        if desvio_ticket > 0:
            resumo["z_ticket"] = (
                (resumo["ticket_medio"] - media_ticket) / desvio_ticket
            ).round(2)
        else:
            resumo["z_ticket"] = 0

    return resumo.sort_values("total_faturado", ascending=False)


def get_resumo_anomalias(
    df_anomalias: pd.DataFrame,
    df_descontos: pd.DataFrame,
) -> dict:
    """Retorna resumo consolidado das anomalias"""
    return {
        "total_anomalias_venda":   len(df_anomalias),
        "total_descontos_anômalos": len(df_descontos),
        "valor_anomalias":         float(df_anomalias["total"].sum()) if not df_anomalias.empty else 0,
        "valor_descontos":         float(df_descontos["desconto"].sum()) if not df_descontos.empty else 0,
    }