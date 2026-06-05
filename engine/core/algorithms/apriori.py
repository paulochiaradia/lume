import pandas as pd
import numpy as np
import logging

log = logging.getLogger(__name__)


def calcular_market_basket(
    df_itens: pd.DataFrame,
    min_support: float = 0.01,
    min_confidence: float = 0.2,
    min_lift: float = 1.2,
) -> pd.DataFrame:
    """
    Calcula associações entre produtos usando o algoritmo Apriori.

    Args:
        df_itens: DataFrame com colunas venda_id e produto_key
        min_support: Suporte mínimo (% de transações que contém o itemset)
        min_confidence: Confiança mínima (P(B|A))
        min_lift: Lift mínimo (força da associação acima do acaso)

    Returns:
        DataFrame com regras de associação ordenadas por lift
    """
    if df_itens.empty:
        log.warning("Apriori: DataFrame vazio")
        return pd.DataFrame()

    try:
        from mlxtend.frequent_patterns import apriori, association_rules
        from mlxtend.preprocessing import TransactionEncoder
    except ImportError:
        log.error("mlxtend não instalado")
        return pd.DataFrame()

    # ── Monta a matriz de transações ─────────────────────────
    # Cada linha é uma venda, cada coluna é um produto (True/False)
    transacoes = df_itens.groupby("venda_id")["produto_key"].apply(list).tolist()

    if len(transacoes) < 10:
        log.warning("Apriori: poucos dados para análise")
        return pd.DataFrame()

    te = TransactionEncoder()
    te_array = te.fit(transacoes).transform(transacoes)
    df_basket = pd.DataFrame(te_array, columns=te.columns_)

    log.info(f"Apriori: {len(transacoes)} transações, {len(te.columns_)} produtos únicos")

    # ── Calcula itemsets frequentes ──────────────────────────
    frequent_itemsets = apriori(
        df_basket,
        min_support=min_support,
        use_colnames=True,
        max_len=3,  # máximo de 3 produtos por regra
    )

    if frequent_itemsets.empty:
        log.warning("Apriori: nenhum itemset frequente encontrado")
        return pd.DataFrame()

    log.info(f"Apriori: {len(frequent_itemsets)} itemsets frequentes")

    # ── Gera regras de associação ────────────────────────────
    rules = association_rules(
        frequent_itemsets,
        metric="lift",
        min_threshold=min_lift,
    )

    if rules.empty:
        log.warning("Apriori: nenhuma regra encontrada com os thresholds definidos")
        return pd.DataFrame()

    # Filtra por confiança mínima
    rules = rules[rules["confidence"] >= min_confidence]

    # ── Formata o resultado ──────────────────────────────────
    rules["antecedents"] = rules["antecedents"].apply(
        lambda x: ", ".join(sorted(list(x)))
    )
    rules["consequents"] = rules["consequents"].apply(
        lambda x: ", ".join(sorted(list(x)))
    )

    resultado = rules[[
        "antecedents", "consequents",
        "support", "confidence", "lift"
    ]].copy()

    resultado["support"]    = (resultado["support"] * 100).round(2)
    resultado["confidence"] = (resultado["confidence"] * 100).round(2)
    resultado["lift"]       = resultado["lift"].round(3)

    resultado = resultado.sort_values("lift", ascending=False).reset_index(drop=True)

    log.info(f"Apriori: {len(resultado)} regras geradas")

    return resultado


def get_sugestoes_por_produto(
    df_regras: pd.DataFrame,
    produto_key: str,
    top_n: int = 5,
) -> pd.DataFrame:
    """
    Retorna os produtos mais frequentemente comprados junto
    com um produto específico.
    """
    if df_regras.empty:
        return pd.DataFrame()

    sugestoes = df_regras[
        df_regras["antecedents"].str.contains(produto_key, na=False)
    ].copy()

    return sugestoes.head(top_n)


def get_top_regras(df_regras: pd.DataFrame, top_n: int = 10) -> pd.DataFrame:
    """Retorna as top N regras por lift"""
    if df_regras.empty:
        return pd.DataFrame()
    return df_regras.head(top_n)