import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import streamlit as st
import pandas as pd
import plotly.express as px
import duckdb
from core.api_client import get_estoque_completo

DUCKDB_PATH = os.getenv("DUCKDB_PATH", "/app/data/duckdb")

def fmt_brl(value: float) -> str:
    return f"R$ {value:,.2f}".replace(",", "X").replace(".", ",").replace("X", ".")

def load_forecast_resumo(client_key: str) -> pd.DataFrame:
    """Lê o resumo do forecast do DuckDB"""
    path = os.path.join(DUCKDB_PATH, f"{client_key}.duckdb")
    if not os.path.exists(path):
        st.warning(f"Arquivo DuckDB não encontrado: {path}")
        return pd.DataFrame()
    try:
        conn = duckdb.connect(path)
        df = conn.execute("SELECT * FROM forecast_resumo").df()
        conn.close()
        return df
    except Exception as e:
        st.warning(f"Erro ao ler forecast: {e}")
        return pd.DataFrame()

def render():
    st.title("📦 Estoque & Compras")
    st.caption("Posição atual de estoque com previsão de demanda e alertas de reposição")

    client_key = st.session_state.get("client_key", "loja_teste")

    with st.spinner("Carregando dados de estoque..."):
        estoque  = get_estoque_completo()
        forecast = load_forecast_resumo(client_key)

    if not estoque:
        st.warning("Nenhum dado de estoque encontrado.")
        return

    df_est = pd.DataFrame(estoque)

    # ── KPIs ─────────────────────────────────────────────────
    st.subheader("Visão Geral")

    total_itens   = len(df_est)
    itens_alerta  = df_est["alerta"].sum()
    valor_estoque = (df_est["quantidade"] * df_est["preco_custo"]).sum()
    cobertura_ok  = total_itens - itens_alerta

    col1, col2, col3, col4 = st.columns(4)
    with col1:
        st.metric("Total de SKUs", str(total_itens))
    with col2:
        st.metric("Valor em Estoque", fmt_brl(valor_estoque))
    with col3:
        st.metric("Estoque OK", str(int(cobertura_ok)))
    with col4:
        st.metric(
            "Alertas de Reposição",
            str(int(itens_alerta)),
            delta_color="inverse"
        )

    st.divider()

    # ── Alertas de reposição ─────────────────────────────────
    df_alerta = df_est[df_est["alerta"] == True].copy()

    if not df_alerta.empty:
        st.subheader("🚨 Produtos que Precisam de Reposição")

        # Cruza com forecast se disponível
        if not forecast.empty:
            df_alerta = df_alerta.merge(
                forecast[["produto_key", "quantidade_prevista"]],
                on="produto_key",
                how="left"
            )
            df_alerta["quantidade_prevista"] = df_alerta["quantidade_prevista"].fillna(0)
            df_alerta["dias_cobertura"] = (
                df_alerta["quantidade"] /
                (df_alerta["quantidade_prevista"] / 30)
            ).replace([float("inf"), float("nan")], 0).round(1)

            df_alerta["pedido_sugerido"] = (
                df_alerta["quantidade_prevista"] - df_alerta["quantidade"]
            ).clip(lower=0).round(0)

        df_tabela = df_alerta[[
            "nome", "categoria", "quantidade",
            "quantidade_min",
        ]].copy()

        if "dias_cobertura" in df_alerta.columns:
            df_tabela["dias_cobertura"]  = df_alerta["dias_cobertura"].apply(
                lambda x: f"{x:.0f} dias"
            )
            df_tabela["pedido_sugerido"] = df_alerta["pedido_sugerido"].apply(
                lambda x: f"{x:.0f} un"
            )

        df_tabela.columns = [
            "Produto", "Categoria", "Qtd Atual", "Qtd Mínima"
        ] + (["Cobertura", "Pedido Sugerido"] if "dias_cobertura" in df_alerta.columns else [])

        st.dataframe(
            df_tabela,
            use_container_width=True,
            hide_index=True
        )

    else:
        st.success("Todos os produtos estão com estoque adequado.")

    st.divider()

    # ── Forecast dos top produtos ────────────────────────────
    if not forecast.empty:
        st.subheader("📈 Previsão de Demanda — Próximos 30 dias")

        df_fc = forecast.merge(
            df_est[["produto_key", "nome", "categoria"]],
            on="produto_key",
            how="left"
        ).head(15)

        fig = px.bar(
            df_fc.sort_values("quantidade_prevista", ascending=True),
            x="quantidade_prevista",
            y="nome",
            orientation="h",
            error_x=df_fc.sort_values("quantidade_prevista", ascending=True)[
                "quantidade_max"
            ] - df_fc.sort_values("quantidade_prevista", ascending=True)[
                "quantidade_prevista"
            ],
            labels={
                "quantidade_prevista": "Quantidade Prevista",
                "nome": "Produto"
            },
            color="categoria",
            height=500,
        )
        fig.update_layout(
            plot_bgcolor="rgba(0,0,0,0)",
            paper_bgcolor="rgba(0,0,0,0)",
        )
        st.plotly_chart(fig, use_container_width=True)

    st.divider()

    # ── Estoque completo ─────────────────────────────────────
    st.subheader("Posição Completa de Estoque")

    col1, col2 = st.columns(2)
    with col1:
        cats = ["Todas"] + sorted(df_est["categoria"].unique().tolist())
        cat_filtro = st.selectbox("Filtrar por categoria", cats)

    with col2:
        alertas_opcoes = ["Todos", "Apenas alertas", "Sem alertas"]
        alerta_filtro  = st.selectbox("Filtrar por status", alertas_opcoes)

    df_f = df_est.copy()
    if cat_filtro != "Todas":
        df_f = df_f[df_f["categoria"] == cat_filtro]
    if alerta_filtro == "Apenas alertas":
        df_f = df_f[df_f["alerta"] == True]
    elif alerta_filtro == "Sem alertas":
        df_f = df_f[df_f["alerta"] == False]

    df_tabela_completa = df_f[[
        "nome", "categoria", "quantidade",
        "quantidade_min", "preco_custo", "preco_venda"
    ]].copy()

    df_tabela_completa["preco_custo"]  = df_tabela_completa["preco_custo"].apply(fmt_brl)
    df_tabela_completa["preco_venda"]  = df_tabela_completa["preco_venda"].apply(fmt_brl)
    df_tabela_completa["quantidade"]   = df_tabela_completa["quantidade"].apply(
        lambda x: f"{x:.0f} un"
    )
    df_tabela_completa["quantidade_min"] = df_tabela_completa["quantidade_min"].apply(
        lambda x: f"{x:.0f} un"
    )

    df_tabela_completa.columns = [
        "Produto", "Categoria", "Qtd Atual",
        "Qtd Mínima", "Preço Custo", "Preço Venda"
    ]

    st.dataframe(df_tabela_completa, use_container_width=True, hide_index=True)

    st.divider()

    # ── Insight ──────────────────────────────────────────────
    st.subheader("💡 Insight")

    if int(itens_alerta) > 0:
        st.warning(
            f"**{int(itens_alerta)} produto(s)** estão abaixo do estoque mínimo. "
            f"Priorize a reposição dos itens classe A para evitar ruptura."
        )
    else:
        st.success("Estoque em níveis adequados. Continue monitorando os produtos classe A.")