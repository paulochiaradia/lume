import pandas as pd
import numpy as np
import logging

log = logging.getLogger(__name__)


def calcular_abc(df_itens: pd.DataFrame,
                 threshold_a: float = 0.80,
                 threshold_b: float = 0.95) -> pd.DataFrame:
    """
    Calcula a Curva ABC dos produtos por faturamento acumulado.

    Args:
        df_itens: DataFrame com colunas produto_key, total
        threshold_a: Percentual acumulado para classe A (padrão 80%)
        threshold_b: Percentual acumulado para classe B (padrão 95%)

    Returns:
        DataFrame com classificação ABC por produto
    """
    if df_itens.empty:
        log.warning("ABC: DataFrame de itens vazio")
        return pd.DataFrame()

    # ── Agrega faturamento por produto ───────────────────────
    abc = df_itens.groupby("produto_key").agg(
        faturamento   = ("total", "sum"),
        quantidade    = ("quantidade", "sum"),
        num_vendas    = ("produto_key", "count"),
    ).reset_index()

    # Ordena por faturamento decrescente
    abc = abc.sort_values("faturamento", ascending=False).reset_index(drop=True)

    # ── Calcula percentual acumulado ─────────────────────────
    total = abc["faturamento"].sum()
    abc["percentual"]         = abc["faturamento"] / total
    abc["percentual_acumulado"] = abc["percentual"].cumsum()

    # ── Classifica em A, B ou C ──────────────────────────────
    abc["classe"] = abc["percentual_acumulado"].apply(
        lambda x: "A" if x <= threshold_a else ("B" if x <= threshold_b else "C")
    )

    # Arredonda
    abc["faturamento"]            = abc["faturamento"].round(2)
    abc["percentual"]             = (abc["percentual"] * 100).round(2)
    abc["percentual_acumulado"]   = (abc["percentual_acumulado"] * 100).round(2)
    abc["quantidade"]             = abc["quantidade"].round(2)

    log.info(f"ABC calculado para {len(abc)} produtos")
    log.info(f"Classe A: {len(abc[abc['classe']=='A'])} | "
             f"Classe B: {len(abc[abc['classe']=='B'])} | "
             f"Classe C: {len(abc[abc['classe']=='C'])}")

    return abc


def calcular_xyz(df_itens: pd.DataFrame) -> pd.DataFrame:
    """
    Calcula a Curva XYZ dos produtos por previsibilidade de demanda.
    X = demanda regular (CV <= 0.5)
    Y = demanda variável (0.5 < CV <= 1.0)
    Z = demanda errática (CV > 1.0)

    Args:
        df_itens: DataFrame com colunas produto_key, quantidade, data_venda

    Returns:
        DataFrame com classificação XYZ por produto
    """
    if df_itens.empty:
        return pd.DataFrame()

    df = df_itens.copy()
    df["data_venda"] = pd.to_datetime(df["data_venda"], utc=True)
    df["mes"] = df["data_venda"].dt.to_period("M")

    # Agrega quantidade por produto por mês
    mensal = df.groupby(["produto_key", "mes"])["quantidade"].sum().reset_index()

    # Calcula coeficiente de variação por produto
    cv = mensal.groupby("produto_key")["quantidade"].agg(
        media  = "mean",
        desvio = "std"
    ).reset_index()

    cv["desvio"] = cv["desvio"].fillna(0)
    cv["cv"]     = cv["desvio"] / cv["media"].replace(0, np.nan)
    cv["cv"]     = cv["cv"].fillna(0)

    # Classifica
    cv["classe_xyz"] = cv["cv"].apply(
        lambda x: "X" if x <= 0.5 else ("Y" if x <= 1.0 else "Z")
    )

    return cv[["produto_key", "media", "desvio", "cv", "classe_xyz"]].round(3)


def calcular_abc_xyz(df_itens: pd.DataFrame,
                     threshold_a: float = 0.80,
                     threshold_b: float = 0.95) -> pd.DataFrame:
    """
    Combina ABC e XYZ em uma única matriz.
    Retorna classificação combinada ex: AX, AY, BX, CZ etc.
    """
    df_abc = calcular_abc(df_itens, threshold_a, threshold_b)
    df_xyz = calcular_xyz(df_itens)

    if df_abc.empty or df_xyz.empty:
        return df_abc

    resultado = df_abc.merge(
        df_xyz[["produto_key", "classe_xyz", "cv"]],
        on="produto_key",
        how="left"
    )

    resultado["classe_xyz"]    = resultado["classe_xyz"].fillna("Z")
    resultado["classe_abc_xyz"] = resultado["classe"] + resultado["classe_xyz"]

    log.info("Matriz ABC/XYZ calculada")
    log.info(f"\n{resultado['classe_abc_xyz'].value_counts().to_string()}")

    return resultado


def get_resumo_abc(df_abc: pd.DataFrame) -> pd.DataFrame:
    """Retorna resumo por classe ABC"""
    return df_abc.groupby("classe").agg(
        produtos      = ("produto_key", "count"),
        faturamento   = ("faturamento", "sum"),
        percentual    = ("percentual", "sum"),
    ).round(2).reset_index()