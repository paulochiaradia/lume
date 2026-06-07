import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import streamlit as st
import pandas as pd
import plotly.express as px
import plotly.graph_objects as go
import duckdb

DUCKDB_PATH = os.getenv("DUCKDB_PATH", "/app/data/duckdb")

def fmt_brl(value: float) -> str:
    return f"R$ {value:,.2f}".replace(",", "X").replace(".", ",").replace("X", ".")

def fmt_pct(value: float) -> str:
    return f"{value:.1f}%"

def load_duckdb(client_key: str, table: str) -> pd.DataFrame:
    path = os.path.join(DUCKDB_PATH, f"{client_key}.duckdb")
    if not os.path.exists(path):
        return pd.DataFrame()
    try:
        conn = duckdb.connect(path)
        df = conn.execute(f"SELECT * FROM {table}").df()
        conn.close()
        return df
    except Exception:
        return pd.DataFrame()

def render():
    st.title("💰 Preço & Margem")
    st.caption("Elasticidade de preço, margem bruta e simulador de precificação")

    client_key = st.session_state.get("client_key", "loja_teste")

    with st.spinner("Carregando análise de preço e margem..."):
        df_elast  = load_duckdb(client_key, "elasticidade_resultado")
        df_margem = load_duckdb(client_key, "margem_resultado")

    if df_elast.empty and df_margem.empty:
        st.warning("Análise de preço ainda não calculada. Aguarde o próximo ciclo do engine.")
        return

    # ── KPIs ─────────────────────────────────────────────────
    st.subheader("Visão Geral")

    col1, col2, col3, col4 = st.columns(4)

    with col1:
        receita_total = df_margem["receita"].sum() if not df_margem.empty else 0
        st.metric("Receita Total", fmt_brl(float(receita_total)))

    with col2:
        margem_total = df_margem["margem_bruta"].sum() if not df_margem.empty else 0
        st.metric("Margem Bruta Total", fmt_brl(float(margem_total)))

    with col3:
        pct_margem = (margem_total / receita_total * 100) if receita_total > 0 else 0
        st.metric("Margem Média", fmt_pct(float(pct_margem)))

    with col4:
        elasticos = len(df_elast[df_elast["tipo"] == "elastico"]) if not df_elast.empty else 0
        st.metric("Produtos Elásticos", str(elasticos))

    st.divider()

    # ── Margem por produto ───────────────────────────────────
    if not df_margem.empty:
        st.subheader("Margem Bruta por Produto")

        col1, col2 = st.columns(2)

        with col1:
            fig_margem = px.bar(
                df_margem.head(15).sort_values("pct_margem", ascending=True),
                x="pct_margem",
                y="nome",
                orientation="h",
                color="pct_margem",
                color_continuous_scale="RdYlGn",
                labels={
                    "pct_margem": "Margem (%)",
                    "nome": "Produto"
                },
                height=500,
            )
            fig_margem.update_layout(
                plot_bgcolor="rgba(0,0,0,0)",
                paper_bgcolor="rgba(0,0,0,0)",
                title="% Margem por Produto",
            )
            st.plotly_chart(fig_margem, use_container_width=True)

        with col2:
            fig_receita = px.bar(
                df_margem.head(15).sort_values("margem_bruta", ascending=True),
                x="margem_bruta",
                y="nome",
                orientation="h",
                color="categoria",
                labels={
                    "margem_bruta": "Margem Bruta (R$)",
                    "nome": "Produto"
                },
                height=500,
            )
            fig_receita.update_layout(
                plot_bgcolor="rgba(0,0,0,0)",
                paper_bgcolor="rgba(0,0,0,0)",
                title="Margem Bruta (R$) por Produto",
            )
            st.plotly_chart(fig_receita, use_container_width=True)

        st.subheader("Detalhamento de Margem")
        df_margem_fmt = df_margem[[
            "nome", "categoria", "receita",
            "custo_total", "margem_bruta", "pct_margem"
        ]].copy()
        df_margem_fmt["receita"]      = df_margem_fmt["receita"].apply(fmt_brl)
        df_margem_fmt["custo_total"]  = df_margem_fmt["custo_total"].apply(fmt_brl)
        df_margem_fmt["margem_bruta"] = df_margem_fmt["margem_bruta"].apply(fmt_brl)
        df_margem_fmt["pct_margem"]   = df_margem_fmt["pct_margem"].apply(fmt_pct)
        df_margem_fmt.columns = [
            "Produto", "Categoria", "Receita",
            "Custo Total", "Margem Bruta", "% Margem"
        ]
        st.dataframe(df_margem_fmt, use_container_width=True, hide_index=True)

    st.divider()

    # ── Elasticidade ─────────────────────────────────────────
    if not df_elast.empty:
        st.subheader("Elasticidade de Preço")

        CORES_TIPO = {
            "elastico":   "#1B5E20",
            "moderado":   "#F57C00",
            "inelastico": "#B71C1C",
        }

        col1, col2 = st.columns(2)

        with col1:
            contagem = df_elast["tipo"].value_counts().reset_index()
            contagem.columns = ["tipo", "quantidade"]
            contagem["label"] = contagem["tipo"].map({
                "elastico":   "Elástico",
                "moderado":   "Moderado",
                "inelastico": "Inelástico",
            })

            fig_pie = px.pie(
                contagem,
                values="quantidade",
                names="label",
                color="tipo",
                color_discrete_map=CORES_TIPO,
                hole=0.4,
                title="Distribuição por Tipo de Elasticidade",
            )
            fig_pie.update_layout(
                plot_bgcolor="rgba(0,0,0,0)",
                paper_bgcolor="rgba(0,0,0,0)",
            )
            st.plotly_chart(fig_pie, use_container_width=True)

        with col2:
            st.subheader("Recomendações por Produto")
            df_elast_fmt = df_elast[[
                "nome", "elasticidade", "tipo", "recomendacao"
            ]].copy()
            df_elast_fmt["elasticidade"] = df_elast_fmt["elasticidade"].apply(
                lambda x: f"{x:.3f}"
            )
            df_elast_fmt.columns = [
                "Produto", "Elasticidade", "Tipo", "Recomendação"
            ]
            st.dataframe(
                df_elast_fmt,
                use_container_width=True,
                hide_index=True,
                height=400,
            )

    st.divider()

    # ── Simulador de precificação ────────────────────────────
    st.subheader("🧮 Simulador de Precificação")
    st.caption("Simule o impacto de uma mudança de preço na receita")

    if not df_elast.empty:
        col1, col2 = st.columns(2)

        with col1:
            produto_selecionado = st.selectbox(
                "Selecione o produto",
                df_elast["produto_key"].tolist(),
                format_func=lambda x: df_elast[
                    df_elast["produto_key"] == x
                ]["nome"].iloc[0]
            )

        with col2:
            variacao = st.slider(
                "Variação de preço (%)",
                min_value=-30,
                max_value=30,
                value=0,
                step=5,
            )

        if produto_selecionado and variacao != 0:
            from core.algorithms.elasticity import simular_preco

            simulacao = simular_preco(df_elast, produto_selecionado, variacao)

            if simulacao:
                col1, col2, col3 = st.columns(3)

                with col1:
                    st.metric(
                        "Preço Atual → Novo",
                        fmt_brl(simulacao["preco_novo"]),
                        f"{variacao:+.0f}%"
                    )
                with col2:
                    st.metric(
                        "Variação na Quantidade",
                        f"{simulacao['variacao_qtd']:+.1f}%"
                    )
                with col3:
                    impacto = simulacao["impacto_receita"]
                    st.metric(
                        "Impacto na Receita",
                        fmt_brl(abs(impacto)),
                        f"{simulacao['impacto_pct']:+.1f}%",
                        delta_color="normal" if impacto > 0 else "inverse"
                    )

    st.divider()

    # ── Insight ──────────────────────────────────────────────
    st.subheader("💡 Insight")

    if not df_margem.empty:
        menor_margem = df_margem.loc[df_margem["pct_margem"].idxmin()]
        maior_margem = df_margem.loc[df_margem["pct_margem"].idxmax()]

        st.info(
            f"**Maior margem:** {maior_margem['nome']} com "
            f"{fmt_pct(maior_margem['pct_margem'])} de margem bruta.  \n"
            f"**Menor margem:** {menor_margem['nome']} com "
            f"{fmt_pct(menor_margem['pct_margem'])} — avalie reposicionamento de preço."
        )