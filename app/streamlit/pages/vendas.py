import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import streamlit as st
import pandas as pd
import plotly.express as px
import plotly.graph_objects as go
from core.api_client import get_vendas_por_dia, get_home_kpis

def fmt_brl(value: float) -> str:
    return f"R$ {value:,.2f}".replace(",", "X").replace(".", ",").replace("X", ".")

def render():
    st.title("📊 Vendas")
    st.caption("Performance comercial com contexto temporal")

    with st.spinner("Carregando dados de vendas..."):
        kpis    = get_home_kpis()
        vendas  = get_vendas_por_dia()

    if not kpis or not vendas:
        st.warning("Nenhum dado de vendas encontrado.")
        return

    # ── KPIs ─────────────────────────────────────────────────
    st.subheader("Resumo Geral")

    col1, col2, col3, col4 = st.columns(4)
    with col1:
        st.metric("Faturamento Total", fmt_brl(kpis["faturamento"]))
    with col2:
        st.metric("Ticket Médio", fmt_brl(kpis["ticket_medio"]))
    with col3:
        st.metric("Total de Vendas", str(kpis["total_vendas"]))
    with col4:
        st.metric("Total de Descontos", fmt_brl(kpis["total_desconto"]))

    st.divider()

    # ── Gráfico de linha — faturamento por dia ───────────────
    df = pd.DataFrame(vendas)

    st.subheader("Evolução do Faturamento")

    fig_linha = px.line(
        df,
        x="dia",
        y="faturamento",
        markers=True,
        labels={
            "dia": "Data",
            "faturamento": "Faturamento (R$)"
        },
    )

    fig_linha.update_traces(
        line_color="#2E7D32",
        marker_color="#2E7D32",
        marker_size=8,
    )

    fig_linha.update_layout(
        plot_bgcolor="rgba(0,0,0,0)",
        paper_bgcolor="rgba(0,0,0,0)",
        hovermode="x unified",
    )

    st.plotly_chart(fig_linha, use_container_width=True)

    st.divider()

    # ── Gráfico de barras — vendas por dia ───────────────────
    st.subheader("Número de Vendas por Dia")

    fig_barras = px.bar(
        df,
        x="dia",
        y="vendas",
        labels={
            "dia": "Data",
            "vendas": "Número de Vendas"
        },
        color_discrete_sequence=["#1565C0"],
    )

    fig_barras.update_layout(
        plot_bgcolor="rgba(0,0,0,0)",
        paper_bgcolor="rgba(0,0,0,0)",
    )

    st.plotly_chart(fig_barras, use_container_width=True)

    st.divider()

    # ── Tabela detalhada ─────────────────────────────────────
    st.subheader("Detalhamento por Dia")

    df_tabela = df.copy()
    df_tabela["faturamento"] = df_tabela["faturamento"].apply(fmt_brl)
    df_tabela["ticket_medio"] = (
        df["faturamento"] / df["vendas"]
    ).apply(fmt_brl)

    df_tabela.columns = ["Data", "Vendas", "Faturamento", "Ticket Médio"]
    df_tabela = df_tabela[["Data", "Vendas", "Faturamento", "Ticket Médio"]]

    st.dataframe(df_tabela, use_container_width=True, hide_index=True)

    st.divider()

    # ── Insight automático ───────────────────────────────────
    st.subheader("💡 Insight")

    melhor_dia = df.loc[df["faturamento"].idxmax()]
    pior_dia   = df.loc[df["faturamento"].idxmin()]

    st.info(
    f"**Melhor dia:** {melhor_dia['dia']} com "
    f"{fmt_brl(melhor_dia['faturamento'])} "
    f"em {int(melhor_dia['vendas'])} venda(s).  \n"
    f"**Pior dia:** {pior_dia['dia']} com "
    f"{fmt_brl(pior_dia['faturamento'])} "
    f"em {int(pior_dia['vendas'])} venda(s)."
)