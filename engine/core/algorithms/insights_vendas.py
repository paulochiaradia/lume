import pandas as pd
import duckdb
import os
import logging
from datetime import datetime, timezone

log = logging.getLogger(__name__)

DUCKDB_PATH = os.getenv("DUCKDB_PATH", "/app/data/duckdb")

def get_duckdb(client_key: str):
    path = os.path.join(DUCKDB_PATH, f"{client_key}.duckdb")
    if not os.path.exists(path):
        return None
    return duckdb.connect(path)

def read_table(client_key: str, table: str) -> pd.DataFrame:
    conn = get_duckdb(client_key)
    if conn is None:
        return pd.DataFrame()
    try:
        df = conn.execute(f"SELECT * FROM {table}").df()
        conn.close()
        return df
    except Exception:
        return pd.DataFrame()

def gerar_insights_vendas(client_key: str, kpis: dict = None) -> list:
    """
    Gera insights EXCLUSIVOS para o painel de Vendas.
    Foco: Vendedores, UPA, Ticket Médio, Descontos e Sazonalidade.
    """
    insights = []

    # ── 1. Anomalia de Vendedor (Descontos ou UPA) ───────────
    try:
        df_vend = read_table(client_key, "anomalias_vendedores")
        if not df_vend.empty:
            vendedor = df_vend["vendedor_id"].iloc[0] if "vendedor_id" in df_vend.columns else "Não Informado"
            insights.append({
                "tipo":       "vendedor_anomalia",
                "prioridade": 1,
                "titulo":     "Atenção com Vendedor",
                "mensagem":   f"O vendedor {vendedor} apresentou métricas atípicas (alto volume de descontos ou UPA muito baixo). Revise as últimas vendas.",
                "acao":       "Ver Ranking",
                "href":       "/vendas",
                "categoria":  "vendas",
                "icone":      "warning",
            })
    except Exception as e:
        log.warning(f"Erro ao gerar insight de anomalia de vendedor: {e}")

    # ── 2. Oportunidade de Ticket Médio (Combo) ──────────────
    try:
        df_basket = read_table(client_key, "basket_resultado")
        if not df_basket.empty:
            top = df_basket.sort_values("lift", ascending=False).iloc[0]
            insights.append({
                "tipo":       "venda_cruzada",
                "prioridade": 2,
                "titulo":     "Oportunidade de UPA",
                "mensagem":   f"Combo forte detectado: Quem leva {top['antecedents']} costuma levar {top['consequents']}. Instrua a equipe a oferecer esta venda cruzada para subir o Ticket Médio.",
                "acao":       "Ver Produtos",
                "href":       "/produtos",
                "categoria":  "vendas",
                "icone":      "success",
            })
    except Exception as e:
        log.warning(f"Erro ao gerar insight de basket para vendas: {e}")

    # ── 3. Performance de Categoria (ABC) ────────────────────
    try:
        df_abc = read_table(client_key, "abc_resultado")
        if not df_abc.empty:
            df_a = df_abc[df_abc["classe"] == "A"]
            if not df_a.empty:
                melhor_cat = df_a["categoria"].value_counts().index[0] if "categoria" in df_a.columns else "Diversos"
                insights.append({
                    "tipo":       "categoria_foco",
                    "prioridade": 3,
                    "titulo":     "Tração de Categoria",
                    "mensagem":   f"A categoria '{melhor_cat}' está puxando o faturamento da curva A. Garanta que a equipe conheça o mix de produtos desta linha.",
                    "acao":       "Ver Mix",
                    "href":       "/vendas",
                    "categoria":  "vendas",
                    "icone":      "success",
                })
    except Exception as e:
        log.warning(f"Erro ao gerar insight de categoria: {e}")

    # ── Aplica rotação temporal (Usando tabela isolada) ──────
    insights.sort(key=lambda x: x["prioridade"])
    insights = aplicar_rotacao_temporal_vendas(insights, client_key)
    top3 = insights[:3]

    return insights, top3

def aplicar_rotacao_temporal_vendas(insights: list, client_key: str) -> list:
    conn = get_duckdb(client_key)
    if conn is None:
        return insights

    try:
        df_historico = conn.execute("""
            SELECT tipo FROM insights_vendas_resultado
            WHERE TRY_CAST(gerado_em AS TIMESTAMP) >= CAST(NOW() AS TIMESTAMP) - INTERVAL '24 hours'
        """).df()
        conn.close()

        tipos_recentes = set(df_historico["tipo"].tolist()) if not df_historico.empty else set()

        if not tipos_recentes:
            return insights

        for insight in insights:
            if insight["tipo"] in tipos_recentes:
                insight["prioridade"] += 10 

        insights.sort(key=lambda x: x["prioridade"])
        return insights
    except Exception as e:
        if conn: conn.close()
        return insights

def salvar_insights_vendas(client_key: str, insights: list, top3: list):
    """Salva no DuckDB em tabelas isoladas da Home"""
    conn = get_duckdb(client_key)
    if conn is None:
        return

    try:
        df_todos = pd.DataFrame(insights)
        df_todos["gerado_em"] = datetime.now(timezone.utc).isoformat()
        conn.execute("DROP TABLE IF EXISTS insights_vendas_resultado")
        conn.execute("CREATE TABLE insights_vendas_resultado AS SELECT * FROM df_todos")

        df_top3 = pd.DataFrame(top3)
        df_top3["gerado_em"] = datetime.now(timezone.utc).isoformat()
        conn.execute("DROP TABLE IF EXISTS insights_vendas_top3")
        conn.execute("CREATE TABLE insights_vendas_top3 AS SELECT * FROM df_top3")

        conn.commit()
    except Exception as e:
        log.error(f"Erro ao salvar insights de vendas no DuckDB: {e}")
    finally:
        conn.close()

def salvar_insights_vendas_postgres(client_key: str, top3: list):
    """Salva no PostgreSQL para a API Go consumir"""
    if not top3:
        return

    try:
        from core.db.postgres import get_engine as get_pg_engine
        from sqlalchemy import text

        engine = get_pg_engine()
        schema = f"client_{client_key}"

        with engine.connect() as conn:
            # Vamos limpar APENAS os da categoria vendas para não apagar os da Home
            conn.execute(text(f"DELETE FROM {schema}.insights_cache WHERE categoria = 'vendas'"))

            for insight in top3:
                conn.execute(text(f"""
                    INSERT INTO {schema}.insights_cache
                        (tipo, prioridade, titulo, mensagem, acao, href, categoria, icone, gerado_em)
                    VALUES
                        (:tipo, :prioridade, :titulo, :mensagem, :acao, :href, :categoria, :icone, NOW())
                """), {
                    "tipo":       insight.get("tipo", ""),
                    "prioridade": insight.get("prioridade", 99),
                    "titulo":     insight.get("titulo", ""),
                    "mensagem":   insight.get("mensagem", ""),
                    "acao":       insight.get("acao", ""),
                    "href":       insight.get("href", ""),
                    "categoria":  "vendas", # Força a categoria
                    "icone":      insight.get("icone", "success"),
                })
            conn.commit()
    except Exception as e:
        log.error(f"Erro ao salvar insights de vendas no PostgreSQL: {e}")