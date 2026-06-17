import logging
from engine.core.algorithms.insights import gerar_insights, salvar_insights_postgres
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
    """Executa análise ABC"""
    from core.algorithms.abc_xyz import calcular_abc
    from segments import get_engine
    engine = get_engine(client_key)
    df_itens = engine.get_itens_venda()
    resultado = calcular_abc(df_itens)
    engine.save("abc_resultado", resultado)
    log.info(f"ABC concluido para {client_key}: {len(resultado)} produtos")

@register("insights")
def handle_insights(client_key: str):
    """Gera insights priorizados baseados nos algoritmos"""
    from core.algorithms.insights import gerar_insights, salvar_insights
    from core.db.postgres import read_query

    # Busca KPIs do PostgreSQL
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
    salvar_insights_postgres(client_key, top3)
    log.info(f"{client_key}: {len(insights)} insights, {len(top3)} no top3")

@register("insights_vendas")   
def handle_insights_vendas(client_key: str):
    """Gera insights priorizados e rotacionados exclusivos para o painel de Vendas"""
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