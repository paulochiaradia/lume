import pandas as pd
import numpy as np
import logging
from sklearn.linear_model import LinearRegression

log = logging.getLogger(__name__)


def calcular_elasticidade(
    df_itens: pd.DataFrame,
    df_produtos: pd.DataFrame,
) -> pd.DataFrame:
    """
    Calcula a elasticidade-preço da demanda por produto.
    
    Elasticidade = % variação na quantidade / % variação no preço
    
    < -1: elástico — reduzir preço aumenta receita total
    > -1: inelástico — reduzir preço reduz receita total
    = -1: elasticidade unitária
    
    Args:
        df_itens: DataFrame com colunas produto_key, preco_unitario, quantidade, data_venda
        df_produtos: DataFrame com colunas produto_key, nome, categoria

    Returns:
        DataFrame com elasticidade por produto
    """
    if df_itens.empty:
        log.warning("Elasticidade: DataFrame vazio")
        return pd.DataFrame()

    df = df_itens.copy()
    df["data_venda"] = pd.to_datetime(df["data_venda"], utc=True)
    df["mes"]        = df["data_venda"].dt.to_period("M")

    resultados = []

    for produto_key in df["produto_key"].unique():
        df_prod = df[df["produto_key"] == produto_key].copy()

        # Agrega por mês
        mensal = df_prod.groupby("mes").agg(
            preco_medio  = ("preco_unitario", "mean"),
            quantidade   = ("quantidade", "sum"),
            receita      = ("total", "sum"),
        ).reset_index()

        if len(mensal) < 6:
            continue

        # Calcula variações percentuais
        mensal["var_preco"] = mensal["preco_medio"].pct_change() * 100
        mensal["var_qtd"]   = mensal["quantidade"].pct_change() * 100

        # Remove nulos e infinitos
        mensal = mensal.replace([np.inf, -np.inf], np.nan).dropna()

        if len(mensal) < 5:
            continue

        # Regressão linear para estimar elasticidade
        X = mensal[["var_preco"]].values
        y = mensal["var_qtd"].values

        try:
            model = LinearRegression()
            model.fit(X, y)
            elasticidade = model.coef_[0]
            r2           = model.score(X, y)
        except Exception:
            continue

        # Classificação
        if elasticidade < -1:
            tipo = "elastico"
            recomendacao = "Reduzir preço pode aumentar receita total"
        elif elasticidade > -0.5:
            tipo = "inelastico"
            recomendacao = "Preço pouco sensível — evite descontos desnecessários"
        else:
            tipo = "moderado"
            recomendacao = "Elasticidade moderada — avalie caso a caso"

        # Busca nome do produto
        nome = produto_key
        categoria = ""
        if not df_produtos.empty:
            prod = df_produtos[df_produtos["produto_key"] == produto_key]
            if not prod.empty:
                nome      = prod["nome"].iloc[0]
                categoria = prod["categoria"].iloc[0] if "categoria" in prod.columns else ""

        resultados.append({
            "produto_key":   produto_key,
            "nome":          nome,
            "categoria":     categoria,
            "elasticidade":  round(elasticidade, 3),
            "r2":            round(r2, 3),
            "tipo":          tipo,
            "recomendacao":  recomendacao,
            "preco_medio":   round(mensal["preco_medio"].mean(), 2),
            "receita_total": round(mensal["receita"].sum(), 2),
        })

    if not resultados:
        log.warning("Elasticidade: nenhum produto com dados suficientes")
        return pd.DataFrame()

    df_result = pd.DataFrame(resultados)
    df_result = df_result.sort_values("receita_total", ascending=False)

    log.info(f"Elasticidade calculada para {len(df_result)} produtos")
    log.info(f"Elasticos: {len(df_result[df_result['tipo']=='elastico'])} | "
             f"Inelasticos: {len(df_result[df_result['tipo']=='inelastico'])} | "
             f"Moderados: {len(df_result[df_result['tipo']=='moderado'])}")

    return df_result


def simular_preco(
    df_elasticidade: pd.DataFrame,
    produto_key: str,
    variacao_pct: float,
) -> dict:
    """
    Simula o impacto de uma variação de preço na receita.
    
    Args:
        df_elasticidade: DataFrame com elasticidade por produto
        produto_key: Produto a simular
        variacao_pct: Variação percentual no preço (ex: -10 para -10%)
    
    Returns:
        Dict com impacto estimado na quantidade e receita
    """
    if df_elasticidade.empty:
        return {}

    prod = df_elasticidade[df_elasticidade["produto_key"] == produto_key]
    if prod.empty:
        return {}

    elasticidade  = prod["elasticidade"].iloc[0]
    preco_atual   = prod["preco_medio"].iloc[0]
    receita_atual = prod["receita_total"].iloc[0]

    var_quantidade = elasticidade * variacao_pct
    novo_preco     = preco_atual * (1 + variacao_pct / 100)
    nova_qtd_pct   = 1 + var_quantidade / 100
    nova_receita   = receita_atual * (1 + variacao_pct / 100) * nova_qtd_pct

    return {
        "produto_key":       produto_key,
        "preco_atual":       round(preco_atual, 2),
        "preco_novo":        round(novo_preco, 2),
        "variacao_preco":    variacao_pct,
        "variacao_qtd":      round(var_quantidade, 2),
        "receita_atual":     round(receita_atual, 2),
        "receita_nova":      round(nova_receita, 2),
        "impacto_receita":   round(nova_receita - receita_atual, 2),
        "impacto_pct":       round((nova_receita - receita_atual) / receita_atual * 100, 2),
    }


def calcular_margem_por_produto(
    df_itens: pd.DataFrame,
    df_produtos: pd.DataFrame,
) -> pd.DataFrame:
    """
    Calcula margem bruta por produto.
    """
    if df_itens.empty or df_produtos.empty:
        return pd.DataFrame()

    # Faturamento por produto
    fat = df_itens.groupby("produto_key").agg(
        receita    = ("total", "sum"),
        quantidade = ("quantidade", "sum"),
        desconto   = ("desconto", "sum"),
    ).reset_index()

    # Junta com custo
    resultado = fat.merge(
        df_produtos[["produto_key", "nome", "categoria",
                     "preco_custo", "preco_venda"]],
        on="produto_key",
        how="left"
    )

    resultado["custo_total"]   = resultado["quantidade"] * resultado["preco_custo"]
    resultado["margem_bruta"]  = resultado["receita"] - resultado["custo_total"]
    resultado["pct_margem"]    = (
        resultado["margem_bruta"] / resultado["receita"] * 100
    ).round(2)

    resultado["receita"]       = resultado["receita"].round(2)
    resultado["custo_total"]   = resultado["custo_total"].round(2)
    resultado["margem_bruta"]  = resultado["margem_bruta"].round(2)

    return resultado.sort_values("margem_bruta", ascending=False)

def salvar_elasticidade_postgres(client_key: str, df_elast: pd.DataFrame):
    """Salva os dados de elasticidade no cache do PostgreSQL"""
    if df_elast is None or df_elast.empty:
        return

    import logging
    log = logging.getLogger(__name__)

    try:
        from core.db.postgres import get_engine as get_pg_engine
        engine = get_pg_engine()
        schema = f"client_{client_key}"
        
        df_elast.to_sql("elasticidade_cache", engine, schema=schema, if_exists="replace", index=False)
        log.info(f"[{client_key}] Elasticidade salva no PostgreSQL (cache).")
    except Exception as e:
        log.error(f"[{client_key}] Erro ao salvar Elasticidade no PostgreSQL: {e}")