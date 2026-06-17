import logging
import os
import sys
import time
from core.algorithms.insights import gerar_insights, salvar_insights, salvar_insights_postgres

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from dotenv import load_dotenv
load_dotenv()

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(name)s %(message)s"
)

log = logging.getLogger(__name__)


def run_analytics_for_client(client_key: str):
    """Roda todos os algoritmos para um cliente"""
    log.info(f"Iniciando analytics para cliente: {client_key}")

    try:
        from segments import get_engine
        engine = get_engine(client_key)

        # ── ABC/XYZ ──────────────────────────────────────────
        log.info(f"{client_key}: calculando ABC/XYZ...")
        from core.algorithms.abc_xyz import calcular_abc_xyz
        df_itens = engine.get_itens_venda()
        df_abc   = calcular_abc_xyz(df_itens)
        if not df_abc.empty:
            engine.save("abc_resultado", df_abc)
            log.info(f"{client_key}: ABC/XYZ salvo — {len(df_abc)} produtos")

        # ── RFM ──────────────────────────────────────────────
        log.info(f"{client_key}: calculando RFM...")
        from core.algorithms.rfm import calcular_rfm
        df_vendas = engine.get_vendas()
        df_rfm    = calcular_rfm(df_vendas)
        if not df_rfm.empty:
            engine.save("rfm_resultado", df_rfm)
            log.info(f"{client_key}: RFM salvo — {len(df_rfm)} clientes")

        # ── Market Basket ────────────────────────────────────
        log.info(f"{client_key}: calculando Market Basket...")
        from core.algorithms.apriori import calcular_market_basket
        df_basket = calcular_market_basket(df_itens)
        if not df_basket.empty:
            engine.save("basket_resultado", df_basket)
            log.info(f"{client_key}: Basket salvo — {len(df_basket)} regras")

        # ── Forecast ─────────────────────────────────────────
        log.info(f"{client_key}: calculando Forecast...")
        from core.algorithms.forecast import calcular_forecast, get_forecast_resumo
        df_forecast = calcular_forecast(df_itens, dias_futuro=90)
        if not df_forecast.empty:
            engine.save("forecast_resultado", df_forecast)
            resumo = get_forecast_resumo(df_forecast, dias=30)
            engine.save("forecast_resumo", resumo)
            log.info(f"{client_key}: Forecast salvo")

        # ── Anomalias ─────────────────────────────────────────
        log.info(f"{client_key}: calculando Anomalias...")
        from core.algorithms.anomaly import (
            detectar_anomalias_vendas,
            detectar_anomalias_desconto,
            detectar_anomalias_por_vendedor,
        )
        df_anom = detectar_anomalias_vendas(df_vendas)
        df_desc = detectar_anomalias_desconto(df_vendas)
        df_vend = detectar_anomalias_por_vendedor(df_vendas)

        if not df_anom.empty:
            engine.save("anomalias_vendas", df_anom)
        if not df_desc.empty:
            engine.save("anomalias_descontos", df_desc)
        if not df_vend.empty:
            engine.save("anomalias_vendedores", df_vend)
        log.info(f"{client_key}: Anomalias salvas")

        # ── Elasticidade ─────────────────────────────────────
        log.info(f"{client_key}: calculando Elasticidade...")
        from core.algorithms.elasticity import (
            calcular_elasticidade,
            calcular_margem_por_produto,
        )
        df_produtos = engine.get_produtos()
        df_elast    = calcular_elasticidade(df_itens, df_produtos)
        df_margem   = calcular_margem_por_produto(df_itens, df_produtos)

        if not df_elast.empty:
            engine.save("elasticidade_resultado", df_elast)
        if not df_margem.empty:
            engine.save("margem_resultado", df_margem)
        log.info(f"{client_key}: Elasticidade salva")

        # ── Insights ──────────────────────────────────────────────
        log.info(f"{client_key}: gerando insights...")
        from core.algorithms.insights import gerar_insights, salvar_insights
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

        insights, top3 = gerar_insights(client_key, kpis)
        salvar_insights(client_key, insights, top3)
        salvar_insights_postgres(client_key, top3)
        log.info(f"{client_key}: {len(insights)} insights, {len(top3)} no top3")

        # ── Insights de Vendas (O Novo Motor) ─────────────────────
        log.info(f"{client_key}: gerando insights exclusivos de vendas...")
        from core.algorithms.insights_vendas import (
            gerar_insights_vendas, 
            salvar_insights_vendas, 
            salvar_insights_vendas_postgres
        )
        
        insights_v, top3_v = gerar_insights_vendas(client_key, kpis)
        salvar_insights_vendas(client_key, insights_v, top3_v)
        salvar_insights_vendas_postgres(client_key, top3_v)
        log.info(f"{client_key}: {len(insights_v)} insights de vendas criados.")

        log.info(f"{client_key}: analytics concluído com sucesso")

    except Exception as e:
        log.error(f"{client_key}: erro no analytics — {e}")


def run_all_clients():
    """Roda analytics para todos os clientes ativos"""
    try:
        from core.db.postgres import get_active_clients
        clients = get_active_clients()
        log.info(f"Clientes ativos: {clients}")
        for client_key in clients:
            run_analytics_for_client(client_key)
    except Exception as e:
        log.error(f"Erro ao buscar clientes: {e}")


def main():
    print("Lume - Engine Worker")
    print("====================")

    log.info("engine worker iniciado")

    # Roda uma vez na inicialização
    log.info("Rodando analytics inicial...")
    run_all_clients()

    # Loop — roda a cada 6 horas
    INTERVALO = 6 * 60 * 60

    while True:
        log.info(f"Próxima execução em {INTERVALO // 3600} horas")
        time.sleep(INTERVALO)
        log.info("Rodando analytics agendado...")
        run_all_clients()


if __name__ == "__main__":
    main()