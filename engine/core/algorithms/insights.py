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


def gerar_insights(client_key: str, kpis: dict = None) -> list:
    """
    Gera lista de insights priorizados baseados nos resultados
    dos algoritmos de ML já calculados no DuckDB.

    Retorna lista ordenada por prioridade (1 = mais urgente).
    """
    insights = []

    # ── 1. Estoque crítico ───────────────────────────────────
    try:
        df_forecast = read_table(client_key, "forecast_resumo")
        df_abc      = read_table(client_key, "abc_resultado")

        if not df_forecast.empty and not df_abc.empty:
            produtos_a = df_abc[df_abc["classe"] == "A"]["produto_key"].tolist()

            for _, row in df_forecast.iterrows():
                produto_key       = row["produto_key"]
                qtd_prevista_30d  = float(row["quantidade_prevista"])
                demanda_diaria    = qtd_prevista_30d / 30 if qtd_prevista_30d > 0 else 0

                if demanda_diaria > 0:
                    is_classe_a = produto_key in produtos_a
                    prioridade  = 1 if is_classe_a else 3

                    insights.append({
                        "tipo":       "estoque_critico",
                        "prioridade": prioridade,
                        "titulo":     "Ruptura iminente" if is_classe_a else "Estoque baixo",
                        "mensagem":   (
                            f"Produto {produto_key} (Classe A) com alta demanda prevista: "
                            f"{qtd_prevista_30d:.0f} unidades nos próximos 30 dias. "
                            f"Verifique o nível de estoque."
                        ) if is_classe_a else (
                            f"Produto {produto_key} com demanda prevista de "
                            f"{qtd_prevista_30d:.0f} unidades nos próximos 30 dias."
                        ),
                        "acao":       "Ver estoque",
                        "href":       "/estoque",
                        "categoria":  "estoque",
                        "icone":      "warning",
                    })
                    break  # Só o mais crítico por vez
    except Exception as e:
        log.warning(f"Erro ao gerar insight de estoque: {e}")

    # ── 2. Clientes VIP em risco de churn ────────────────────
    try:
        df_rfm = read_table(client_key, "rfm_resultado")

        if not df_rfm.empty:
            vip_risco = df_rfm[
                (df_rfm["segmento"].isin(["em_risco", "hibernando"])) &
                (df_rfm["valor_total"] > df_rfm["valor_total"].quantile(0.75))
            ].sort_values("valor_total", ascending=False)

            if not vip_risco.empty:
                n        = len(vip_risco)
                valor    = vip_risco["valor_total"].sum()
                recencia = int(vip_risco["recencia"].mean())

                insights.append({
                    "tipo":       "churn_vip",
                    "prioridade": 2,
                    "titulo":     "Clientes VIP sumindo",
                    "mensagem":   (
                        f"{n} cliente(s) de alto valor sem comprar há em média {recencia} dias. "
                        f"Valor histórico em risco: R$ {valor:,.0f}. "
                        f"Entre em contato esta semana."
                    ),
                    "acao":       "Ver clientes",
                    "href":       "/clientes",
                    "categoria":  "clientes",
                    "icone":      "danger",
                })
    except Exception as e:
        log.warning(f"Erro ao gerar insight de churn: {e}")

    # ── 3. Anomalias detectadas ──────────────────────────────
    try:
        df_desc = read_table(client_key, "anomalias_descontos")

        if not df_desc.empty:
            n_anomalias = len(df_desc)
            vendedor    = df_desc["vendedor_id"].value_counts().index[0] if "vendedor_id" in df_desc.columns else "N/A"
            valor       = float(df_desc["desconto"].sum())

            insights.append({
                "tipo":       "anomalia_desconto",
                "prioridade": 3,
                "titulo":     "Descontos anômalos",
                "mensagem":   (
                    f"{n_anomalias} venda(s) com desconto fora do padrão detectadas. "
                    f"Vendedor com mais ocorrências: {vendedor}. "
                    f"Total em descontos incomuns: R$ {valor:,.0f}."
                ),
                "acao":       "Ver anomalias",
                "href":       "/anomalias",
                "categoria":  "anomalias",
                "icone":      "danger",
            })
    except Exception as e:
        log.warning(f"Erro ao gerar insight de anomalias: {e}")

    # ── 4. Oportunidade de combo (Market Basket) ─────────────
    try:
        df_basket = read_table(client_key, "basket_resultado")

        if not df_basket.empty:
            top = df_basket.sort_values("lift", ascending=False).iloc[0]

            insights.append({
                "tipo":       "oportunidade_combo",
                "prioridade": 4,
                "titulo":     "Oportunidade de venda cruzada",
                "mensagem":   (
                    f"Clientes que compram {top['antecedents']} têm "
                    f"{top['confidence']:.0f}% de chance de comprar {top['consequents']}. "
                    f"Treine o balconista para sugerir este combo."
                ),
                "acao":       "Ver produtos",
                "href":       "/produtos",
                "categoria":  "produtos",
                "icone":      "success",
            })
    except Exception as e:
        log.warning(f"Erro ao gerar insight de basket: {e}")

    # ── 5. Produto ABC-A com margem alta ─────────────────────
    try:
        df_margem = read_table(client_key, "margem_resultado")
        df_abc    = read_table(client_key, "abc_resultado")

        if not df_margem.empty and not df_abc.empty:
            produtos_a   = df_abc[df_abc["classe"] == "A"]["produto_key"].tolist()
            margem_a     = df_margem[df_margem["produto_key"].isin(produtos_a)]

            if not margem_a.empty:
                melhor      = margem_a.sort_values("pct_margem", ascending=False).iloc[0]
                nome        = melhor.get("nome", melhor["produto_key"])
                pct         = float(melhor["pct_margem"])
                receita     = float(melhor["receita"])

                insights.append({
                    "tipo":       "margem_alta",
                    "prioridade": 5,
                    "titulo":     "Produto com melhor margem",
                    "mensagem":   (
                        f"{nome} é seu produto mais rentável: {pct:.1f}% de margem bruta "
                        f"com R$ {receita:,.0f} em receita. "
                        f"Priorize seu estoque e visibilidade na loja."
                    ),
                    "acao":       "Ver margens",
                    "href":       "/preco",
                    "categoria":  "financeiro",
                    "icone":      "success",
                })
    except Exception as e:
        log.warning(f"Erro ao gerar insight de margem: {e}")

    # ── 6. Performance positiva ──────────────────────────────
    try:
        if kpis and kpis.get("faturamento", 0) > 0:
            ticket = kpis.get("ticket_medio", 0)
            total  = kpis.get("total_vendas", 0)
            fat    = kpis.get("faturamento", 0)

            insights.append({
                "tipo":       "performance",
                "prioridade": 6,
                "titulo":     "Resumo de performance",
                "mensagem":   (
                    f"{total} vendas com ticket médio de R$ {ticket:,.0f}. "
                    f"Faturamento total: R$ {fat:,.0f}. "
                    f"Produtos classe A respondem por ~80% deste resultado."
                ),
                "acao":       "Ver vendas",
                "href":       "/vendas",
                "categoria":  "vendas",
                "icone":      "success",
            })
    except Exception as e:
        log.warning(f"Erro ao gerar insight de performance: {e}")

    # ── Ordena por prioridade ────────────────────────────────
    insights.sort(key=lambda x: x["prioridade"])

    # ── Aplica rotação temporal ──────────────────────────────
    insights = aplicar_rotacao_temporal(insights, client_key)

    # ── Seleciona top 3 com diversidade de categoria ─────────
    top3 = selecionar_top3_diverso(insights)

    log.info(f"Insights gerados para {client_key}: {len(insights)} total, {len(top3)} selecionados")

    # Retorna todos ordenados — a seleção do top3 fica no salvamento
    return insights, top3

def selecionar_top3_diverso(insights: list) -> list:
    """
    Seleciona os 3 insights mais relevantes garantindo
    diversidade de categoria — nunca 2 cards da mesma categoria.
    """
    selecionados     = []
    categorias_usadas = set()

    for insight in insights:  # já ordenados por prioridade
        categoria = insight.get("categoria", "outros")
        if categoria not in categorias_usadas:
            selecionados.append(insight)
            categorias_usadas.add(categoria)
        if len(selecionados) == 3:
            break

    # Se não chegou a 3 com diversidade, completa com os restantes
    if len(selecionados) < 3:
        for insight in insights:
            if insight not in selecionados:
                selecionados.append(insight)
            if len(selecionados) == 3:
                break

    return selecionados


def aplicar_rotacao_temporal(insights: list, client_key: str) -> list:
    """
    Rebaixa a prioridade de insights que já apareceram
    nas últimas 24h para forçar rotação de conteúdo.
    Lê o histórico do DuckDB e penaliza repetições.
    """
    conn = get_duckdb(client_key)
    if conn is None:
        return insights

    try:
        # Busca os tipos que apareceram nas últimas 24h
        df_historico = conn.execute("""
            SELECT tipo FROM insights_resultado
            WHERE TRY_CAST(gerado_em AS TIMESTAMP) >= CAST(NOW() AS TIMESTAMP) - INTERVAL '24 hours'
        """).df()
        conn.close()

        tipos_recentes = set(df_historico["tipo"].tolist()) if not df_historico.empty else set()

        if not tipos_recentes:
            return insights

        # Penaliza insights repetidos — aumenta prioridade (número maior = menos urgente)
        for insight in insights:
            if insight["tipo"] in tipos_recentes:
                insight["prioridade"] += 10  # rebaixa na ordenação

        # Reordena após penalização
        insights.sort(key=lambda x: x["prioridade"])

        log.info(f"Rotação temporal aplicada — {len(tipos_recentes)} tipos penalizados")
        return insights

    except Exception as e:
        log.warning(f"Erro na rotação temporal: {e}")
        if conn:
            conn.close()
        return insights

def salvar_insights(client_key: str, insights: list, top3: list):
    """Salva todos os insights no DuckDB e o top3 separado"""
    conn = get_duckdb(client_key)
    if conn is None:
        return

    try:
        from datetime import datetime, timezone

        # Salva todos para histórico
        df_todos = pd.DataFrame(insights)
        df_todos["gerado_em"] = datetime.now(timezone.utc).isoformat()
        conn.execute("DROP TABLE IF EXISTS insights_resultado")
        conn.execute("CREATE TABLE insights_resultado AS SELECT * FROM df_todos")

        # Salva top3 separado para o frontend
        df_top3 = pd.DataFrame(top3)
        df_top3["gerado_em"] = datetime.now(timezone.utc).isoformat()
        conn.execute("DROP TABLE IF EXISTS insights_top3")
        conn.execute("CREATE TABLE insights_top3 AS SELECT * FROM df_top3")

        conn.commit()
        log.info(f"Insights salvos no DuckDB para {client_key}")
    except Exception as e:
        log.error(f"Erro ao salvar insights no DuckDB: {e}")
    finally:
        conn.close()


def salvar_insights_postgres(client_key: str, top3: list):
    """Salva top3 no PostgreSQL para a API Go ler"""
    if not top3:
        return

    try:
        from core.db.postgres import get_engine as get_pg_engine
        from sqlalchemy import text

        engine = get_pg_engine()
        schema = f"client_{client_key}"

        with engine.connect() as conn:
            conn.execute(text(f"DELETE FROM {schema}.insights_cache"))

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
                    "categoria":  insight.get("categoria", ""),
                    "icone":      insight.get("icone", "success"),
                })
            conn.commit()
            log.info(f"Top3 insights salvos no PostgreSQL para {client_key}")
    except Exception as e:
        log.error(f"Erro ao salvar insights no PostgreSQL: {e}")