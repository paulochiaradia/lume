import pandas as pd
import numpy as np
from datetime import datetime
import logging

log = logging.getLogger(__name__)


def _score_qcut(series: pd.Series, ascending: bool = True) -> pd.Series:
    """
    Cria scores de 1 a 5 usando qcut com tratamento de bins duplicados.
    ascending=True: maior valor = score maior
    ascending=False: menor valor = score maior (para recência)
    """
    ranked = series.rank(method="first")
    try:
        result = pd.qcut(ranked, q=5, labels=False, duplicates="drop")
        result = result + 1
    except Exception:
        result = pd.Series(
            pd.cut(ranked, bins=5, labels=False),
            index=series.index
        ).fillna(0) + 1

    result = result.fillna(1).astype(int)
    result = result.clip(1, 5)

    if not ascending:
        result = 6 - result

    return result


def calcular_rfm(df_vendas: pd.DataFrame, data_referencia: datetime = None) -> pd.DataFrame:
    """
    Calcula RFM para cada cliente.

    Args:
        df_vendas: DataFrame com colunas data_venda, cliente_id, total
        data_referencia: Data base para cálculo de recência (padrão: hoje)

    Returns:
        DataFrame com scores RFM e segmento por cliente
    """
    if df_vendas.empty:
        log.warning("RFM: DataFrame de vendas vazio")
        return pd.DataFrame()

    if data_referencia is None:
        data_referencia = datetime.now()

    df_vendas = df_vendas.copy()
    df_vendas["data_venda"] = pd.to_datetime(df_vendas["data_venda"], utc=True)
    data_referencia = pd.Timestamp(data_referencia, tz="UTC")

    # ── Calcula métricas base ────────────────────────────────
    rfm = df_vendas.groupby("cliente_id").agg(
        ultima_compra=("data_venda", "max"),
        frequencia=("data_venda", "count"),
        valor_total=("total", "sum")
    ).reset_index()

    rfm["recencia"] = (data_referencia - rfm["ultima_compra"]).dt.days

    # ── Scores de 1 a 5 ─────────────────────────────────────
    rfm["score_r"] = _score_qcut(rfm["recencia"], ascending=False)
    rfm["score_f"] = _score_qcut(rfm["frequencia"], ascending=True)
    rfm["score_m"] = _score_qcut(rfm["valor_total"], ascending=True)

    rfm["score_rfm"] = (
        rfm["score_r"].astype(str) +
        rfm["score_f"].astype(str) +
        rfm["score_m"].astype(str)
    )

    rfm["score_total"] = rfm["score_r"] + rfm["score_f"] + rfm["score_m"]

    # ── Segmentação ──────────────────────────────────────────
    rfm["segmento"] = rfm.apply(_classificar_segmento, axis=1)

    # ── Organiza colunas finais ──────────────────────────────
    resultado = rfm[[
        "cliente_id", "recencia", "frequencia", "valor_total",
        "score_r", "score_f", "score_m", "score_rfm",
        "score_total", "segmento"
    ]].copy()

    resultado["recencia"]    = resultado["recencia"].astype(int)
    resultado["frequencia"]  = resultado["frequencia"].astype(int)
    resultado["valor_total"] = resultado["valor_total"].round(2)

    log.info(f"RFM calculado para {len(resultado)} clientes")
    log.info(f"Distribuicao de segmentos:\n{resultado['segmento'].value_counts().to_string()}")

    return resultado


def _classificar_segmento(row) -> str:
    """
    Classifica o cliente em segmento baseado nos scores RFM.
    Lógica específica para o varejo de materiais de construção.
    """
    r = row["score_r"]
    f = row["score_f"]
    m = row["score_m"]

    if r >= 4 and f >= 4 and m >= 4:
        return "campeon"

    if f >= 3 and m >= 3:
        return "fiel"

    if r <= 2 and f >= 3:
        return "em_risco"

    if r == 1 and f <= 2:
        return "perdido"

    if r >= 4 and f <= 2:
        return "promissor"

    if r >= 3 and f == 1:
        return "novo"

    if r <= 2 and f >= 2:
        return "hibernando"

    return "outros"


def get_clientes_em_risco(df_rfm: pd.DataFrame, limite: int = 20) -> pd.DataFrame:
    """
    Retorna os clientes com maior risco de churn.
    Prioriza quem tinha alto valor mas está sumindo.
    """
    em_risco = df_rfm[
        df_rfm["segmento"].isin(["em_risco", "hibernando"])
    ].copy()

    em_risco = em_risco.sort_values("valor_total", ascending=False)

    return em_risco.head(limite)


def get_resumo_segmentos(df_rfm: pd.DataFrame) -> pd.DataFrame:
    """Retorna resumo agregado por segmento"""
    return df_rfm.groupby("segmento").agg(
        clientes=("cliente_id", "count"),
        valor_medio=("valor_total", "mean"),
        valor_total=("valor_total", "sum"),
        recencia_media=("recencia", "mean"),
    ).round(2).reset_index().sort_values("valor_total", ascending=False)