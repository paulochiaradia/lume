import logging
from worker.job_runner import register

log = logging.getLogger(__name__)

@register("rfm")
def handle_rfm(client_key: str):
    """Executa análise RFM"""
    from core.algorithms.rfm import calcular_rfm
    from segments import get_engine
    engine = get_engine(client_key)
    df_vendas = engine.get_vendas()
    resultado = calcular_rfm(df_vendas)
    engine.save("rfm_resultado", resultado)
    log.info(f"RFM concluido para {client_key}: {len(resultado)} clientes")

@register("abc_xyz")
def handle_abc(client_key: str):
    """Executa análise ABC/XYZ e salva cache no Postgres"""
    from core.algorithms.abc_xyz import calcular_abc_xyz, salvar_abc_postgres
    from segments import get_engine
    engine = get_engine(client_key)
    df_itens = engine.get_itens_venda()
    
    resultado = calcular_abc_xyz(df_itens)
    if not resultado.empty:
        engine.save("abc_resultado", resultado)
        salvar_abc_postgres(client_key, resultado)
        
    log.info(f"ABC/XYZ concluido para {client_key}: {len(resultado)} produtos")

@register("market_basket")
def handle_market_basket(client_key: str):
    """Executa análise de Market Basket (Apriori) e salva cache no Postgres"""
    from core.algorithms.apriori import calcular_market_basket, salvar_basket_postgres
    from segments import get_engine
    engine = get_engine(client_key)
    df_itens = engine.get_itens_venda()
    
    resultado = calcular_market_basket(df_itens)
    if not resultado.empty:
        engine.save("basket_resultado", resultado)
        salvar_basket_postgres(client_key, resultado)
        
    log.info(f"Market Basket concluido para {client_key}: {len(resultado)} regras")

@register("elasticidade")
def handle_elasticidade(client_key: str):
    """Executa análise de Elasticidade/Margem e salva cache no Postgres"""
    from core.algorithms.elasticity import calcular_elasticidade, calcular_margem_por_produto, salvar_elasticidade_postgres
    from segments import get_engine
    engine = get_engine(client_key)
    df_itens = engine.get_itens_venda()
    df_produtos = engine.get_produtos()
    
    df_elast = calcular_elasticidade(df_itens, df_produtos)
    df_margem = calcular_margem_por_produto(df_itens, df_produtos)
    
    if not df_elast.empty:
        engine.save("elasticidade_resultado", df_elast)
        salvar_elasticidade_postgres(client_key, df_elast)
        
    if not df_margem.empty:
        engine.save("margem_resultado", df_margem)
        
    log.info(f"Elasticidade e Margem concluidas para {client_key}")

@register("insights")
def handle_insights(client_key: str):
    """Gera insights priorizados baseados nos algoritmos (Painel Home)"""
    from core.algorithms.insights import gerar_insights, salvar_insights, salvar_insights_postgres
    from core.db.postgres import read_query

    try:
        df_kpis = read_query(client_key, f"""
            SELECT
                COALESCE(SUM(total), 0)  AS faturamento,
                COALESCE(AVG(total), 0)  AS ticket_medio,
                COUNT(*)                 AS total_vendas,
                COALESCE(SUM(desconto), 0) AS total_desconto
            FROM client_{client_key}.vendas
            WHERE status = 'concluida'
        """)
        kpis = df_kpis.iloc[0].to_dict() if not df_kpis.empty else {}
    except Exception:
        kpis = {}

    insights, top3 = gerar_insights(client_key, kpis)
    salvar_insights(client_key, insights, top3)
    salvar_insights_postgres(client_key, insights)
    log.info(f"{client_key}: {len(insights)} insights gerados na Home")

@register("insights_vendas")
def handle_insights_vendas(client_key: str):
    """Gera insights priorizados exclusivos para o painel de Vendas"""
    from core.algorithms.insights_vendas import gerar_insights_vendas, salvar_insights_vendas, salvar_insights_vendas_postgres
    from core.db.postgres import read_query

    try:
        df_kpis = read_query(client_key, f"""
            SELECT
                COALESCE(SUM(total), 0)    AS faturamento,
                COALESCE(AVG(total), 0)    AS ticket_medio,
                COUNT(*)                   AS total_vendas,
                COALESCE(SUM(desconto), 0) AS total_desconto
            FROM client_{client_key}.vendas
            WHERE status = 'concluida'
        """)
        kpis = df_kpis.iloc[0].to_dict() if not df_kpis.empty else {}
    except Exception:
        kpis = {}

    insights, top3 = gerar_insights_vendas(client_key, kpis)
    salvar_insights_vendas(client_key, insights, top3)
    salvar_insights_vendas_postgres(client_key, top3)
    log.info(f"Insights de Vendas concluidos para {client_key}: {len(insights)} gerados, {len(top3)} no top3")