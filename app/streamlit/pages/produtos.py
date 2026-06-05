import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import streamlit as st
import pandas as pd
import plotly.express as px
import duckdb

DUCKDB_PATH = os.getenv("DUCKDB_PATH", "/app/data/duckdb")

CORES_CLASSE = {
    "A": "#1B5E20",
    "B": "#1565C0",
    "C": "#B71C1C",
}

CORES_XYZ = {
    "X": "#1B5E20",
    "Y": "#F57C00",
    "Z": "#B71C1C",
}

def fmt_brl(value: float) -> str:
    return f"R$ {value:,.2f}".replace(",", "X").replace(".", ",").replace("X", ".")

def load_abc_resultado(client_key: str) -> pd.DataFrame:
    """Lê o resultado ABC/XYZ do DuckDB"""
    path = os.path.join(DUCKDB_PATH, f"{client_key}.duckdb")
    if not os.path.exists(path):
        return pd.DataFrame()
    try:
        conn = duckdb.connect(path, read_only=True)
        df = conn.execute("SELECT * FROM abc_resultado").df()
        conn.close()
        return df
    except Exception:
        return pd.DataFrame()

def render():
    st.title("🛒 Produtos & Mix")
    st.caption("Matriz ABC/XYZ — classificação por faturamento e previsibilidade de demanda")

    client_key = st.session_state.get("client_key", "loja_teste")

    with st.spinner("Carregando análise ABC/XYZ..."):
        df = load_abc_resultado(client_key)

    if df.empty:
        st.warning("Análise ABC/XYZ ainda não calculada. Aguarde o próximo ciclo do engine.")
        return

    # ── Filtros ──────────────────────────────────────────────
    col1, col2, col3 = st.columns(3)

    with col1:
        classes = ["Todas"] + sorted(df["classe"].unique().tolist())
        classe_filtro = st.selectbox("Classe ABC", classes)

    with col2:
        if "classe_xyz" in df.columns:
            xyz = ["Todas"] + sorted(df["classe_xyz"].dropna().unique().tolist())
            xyz_filtro = st.selectbox("Classe XYZ", xyz)
        else:
            xyz_filtro = "Todas"

    with col3:
        if "categoria" in df.columns:
            cats = ["Todas"] + sorted(df["categoria"].dropna().unique().tolist())
        else:
            cats = ["Todas"]
        cat_filtro = st.selectbox("Categoria", cats)

    # Aplica filtros
    df_f = df.copy()
    if classe_filtro != "Todas":
        df_f = df_f[df_f["classe"] == classe_filtro]
    if xyz_filtro != "Todas" and "classe_xyz" in df_f.columns:
        df_f = df_f[df_f["classe_xyz"] == xyz_filtro]
    if cat_filtro != "Todas" and "categoria" in df_f.columns:
        df_f = df_f[df_f["categoria"] == cat_filtro]

    st.divider()

    # ── KPIs por classe ABC ──────────────────────────────────
    st.subheader("Distribuição ABC")

    col1, col2, col3 = st.columns(3)
    for classe, col in zip(["A", "B", "C"], [col1, col2, col3]):
        df_cls = df[df["classe"] == classe]
        with col:
            st.metric(
                f"Classe {classe}",
                f"{len(df_cls)} produtos",
                f"{df_cls['percentual'].sum():.1f}% do faturamento"
            )

    st.divider()

    # ── Gráficos ─────────────────────────────────────────────
    col1, col2 = st.columns(2)

    with col1:
        st.subheader("Faturamento por Produto")
        fig = px.bar(
            df_f.sort_values("faturamento", ascending=True).tail(20),
            x="faturamento",
            y="produto_key",
            color="classe",
            color_discrete_map=CORES_CLASSE,
            orientation="h",
            labels={"faturamento": "Faturamento (R$)", "produto_key": "Produto"},
            height=500,
        )
        fig.update_layout(
            plot_bgcolor="rgba(0,0,0,0)",
            paper_bgcolor="rgba(0,0,0,0)",
        )
        st.plotly_chart(fig, use_container_width=True)

    with col2:
        if "classe_xyz" in df.columns:
            st.subheader("Matriz ABC × XYZ")
            matrix = df.groupby(
                ["classe", "classe_xyz"]
            )["produto_key"].count().reset_index()
            matrix.columns = ["ABC", "XYZ", "Produtos"]

            fig_matrix = px.density_heatmap(
                matrix,
                x="XYZ",
                y="ABC",
                z="Produtos",
                color_continuous_scale="Greens",
                labels={"Produtos": "Qtd Produtos"},
            )
            fig_matrix.update_layout(
                plot_bgcolor="rgba(0,0,0,0)",
                paper_bgcolor="rgba(0,0,0,0)",
            )
            st.plotly_chart(fig_matrix, use_container_width=True)

            st.caption(
                "**AX** = Alto valor, demanda previsível — nunca pode faltar.  \n"
                "**AZ** = Alto valor, demanda errática — cuidado com overstock.  \n"
                "**CZ** = Baixo valor, demanda errática — candidato a descontinuar."
            )

    st.divider()

    # ── Tabela detalhada ─────────────────────────────────────
    st.subheader("Detalhamento")

    colunas = ["produto_key", "faturamento", "quantidade",
               "num_vendas", "percentual", "percentual_acumulado", "classe"]

    if "classe_xyz" in df_f.columns:
        colunas.append("classe_xyz")
    if "classe_abc_xyz" in df_f.columns:
        colunas.append("classe_abc_xyz")

    df_tabela = df_f[colunas].copy()
    df_tabela["faturamento"] = df_tabela["faturamento"].apply(fmt_brl)
    df_tabela["percentual"]  = df_tabela["percentual"].apply(lambda x: f"{x:.1f}%")

    st.dataframe(df_tabela, use_container_width=True, hide_index=True)

    st.divider()

    # ── Insight ──────────────────────────────────────────────
    st.subheader("💡 Insight")

    df_a  = df[df["classe"] == "A"]
    pct_a = df_a["percentual"].sum()

    insight = (
        f"**{len(df_a)} produto(s) classe A** representam "
        f"**{pct_a:.1f}%** do faturamento total. "
    )

    if "classe_xyz" in df.columns:
        ax = len(df[(df["classe"] == "A") & (df["classe_xyz"] == "X")])
        az = len(df[(df["classe"] == "A") & (df["classe_xyz"] == "Z")])
        insight += (
            f"Desses, **{ax} são AX** (alto valor e demanda previsível — prioridade máxima de estoque) "
            f"e **{az} são AZ** (alto valor mas demanda errática — evite overstock)."
        )

    st.info(insight)