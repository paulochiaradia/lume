import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import streamlit as st
import pandas as pd
import plotly.express as px
from core.api_client import get_produtos_abc

def render():
    st.title("🛒 Produtos & Mix")
    st.caption("Classificação ABC baseada em faturamento acumulado")

    with st.spinner("Carregando produtos..."):
        produtos = get_produtos_abc()

    if not produtos:
        st.warning("Nenhum dado de produto encontrado.")
        return

    df = pd.DataFrame(produtos)

    # ── Filtros ──────────────────────────────────────────────
    col1, col2 = st.columns(2)

    with col1:
        classes = ["Todas"] + sorted(df["classe"].unique().tolist())
        classe_filtro = st.selectbox("Filtrar por classe", classes)

    with col2:
        categorias = ["Todas"] + sorted(df["categoria"].unique().tolist())
        categoria_filtro = st.selectbox("Filtrar por categoria", categorias)

    # Aplica filtros
    df_filtrado = df.copy()
    if classe_filtro != "Todas":
        df_filtrado = df_filtrado[df_filtrado["classe"] == classe_filtro]
    if categoria_filtro != "Todas":
        df_filtrado = df_filtrado[df_filtrado["categoria"] == categoria_filtro]

    st.divider()

    # ── Métricas por classe ──────────────────────────────────
    st.subheader("Distribuição por Classe")

    col1, col2, col3 = st.columns(3)

    df_a = df[df["classe"] == "A"]
    df_b = df[df["classe"] == "B"]
    df_c = df[df["classe"] == "C"]

    with col1:
        st.metric(
            "Classe A — Alto valor",
            f"{len(df_a)} produtos",
            f"R$ {df_a['faturamento'].sum():,.2f}".replace(",", "X").replace(".", ",").replace("X", ".")
        )
    with col2:
        st.metric(
            "Classe B — Médio valor",
            f"{len(df_b)} produtos",
            f"R$ {df_b['faturamento'].sum():,.2f}".replace(",", "X").replace(".", ",").replace("X", ".")
        )
    with col3:
        st.metric(
            "Classe C — Baixo valor",
            f"{len(df_c)} produtos",
            f"R$ {df_c['faturamento'].sum():,.2f}".replace(",", "X").replace(".", ",").replace("X", ".")
        )

    st.divider()

    # ── Gráfico de barras ────────────────────────────────────
    st.subheader("Faturamento por Produto")

    color_map = {"A": "#2E7D32", "B": "#F57C00", "C": "#C62828"}

    fig = px.bar(
        df_filtrado.sort_values("faturamento", ascending=True),
        x="faturamento",
        y="nome",
        color="classe",
        color_discrete_map=color_map,
        orientation="h",
        labels={
            "faturamento": "Faturamento (R$)",
            "nome": "Produto",
            "classe": "Classe"
        },
        height=max(400, len(df_filtrado) * 35)
    )

    fig.update_layout(
        plot_bgcolor="rgba(0,0,0,0)",
        paper_bgcolor="rgba(0,0,0,0)",
        showlegend=True,
    )

    st.plotly_chart(fig, use_container_width=True)

    st.divider()

    # ── Tabela detalhada ─────────────────────────────────────
    st.subheader("Detalhamento")

    df_tabela = df_filtrado.copy()
    df_tabela["faturamento"] = df_tabela["faturamento"].apply(
        lambda x: f"R$ {x:,.2f}".replace(",", "X").replace(".", ",").replace("X", ".")
    )
    df_tabela.columns = ["Código", "Nome", "Categoria", "Faturamento", "Classe"]

    st.dataframe(
        df_tabela,
        use_container_width=True,
        hide_index=True
    )

    # ── Insight automático ───────────────────────────────────
    st.divider()
    st.subheader("💡 Insight")

    total_fat = df["faturamento"].sum()
    fat_a = df_a["faturamento"].sum()
    pct_a = (fat_a / total_fat * 100) if total_fat > 0 else 0

    st.info(
        f"**{len(df_a)} produto(s) classe A** representam "
        f"**{pct_a:.1f}%** do faturamento total. "
        f"Garanta que esses produtos nunca fiquem sem estoque."
    )