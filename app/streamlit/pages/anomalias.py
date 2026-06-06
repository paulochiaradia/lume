import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import streamlit as st
import pandas as pd
import plotly.express as px
import duckdb

DUCKDB_PATH = os.getenv("DUCKDB_PATH", "/app/data/duckdb")

def fmt_brl(value: float) -> str:
    return f"R$ {value:,.2f}".replace(",", "X").replace(".", ",").replace("X", ".")

def load_duckdb(client_key: str, table: str) -> pd.DataFrame:
    path = os.path.join(DUCKDB_PATH, f"{client_key}.duckdb")
    if not os.path.exists(path):
        return pd.DataFrame()
    try:
        conn = duckdb.connect(path)
        df = conn.execute(f"SELECT * FROM {table}").df()
        conn.close()
        return df
    except Exception as e:
        return pd.DataFrame()

def render():
    st.title("🔍 Anomalias & Vigilância")
    st.caption("Detecção automática de padrões fora do normal em vendas, descontos e vendedores")

    client_key = st.session_state.get("client_key", "loja_teste")

    with st.spinner("Carregando análise de anomalias..."):
        df_anom = load_duckdb(client_key, "anomalias_vendas")
        df_desc = load_duckdb(client_key, "anomalias_descontos")
        df_vend = load_duckdb(client_key, "anomalias_vendedores")

    # ── KPIs ─────────────────────────────────────────────────
    st.subheader("Visão Geral")

    col1, col2, col3, col4 = st.columns(4)
    with col1:
        st.metric(
            "Vendas Anômalas",
            str(len(df_anom)),
            delta_color="inverse"
        )
    with col2:
        valor_anom = df_anom["total"].sum() if not df_anom.empty else 0
        st.metric("Valor em Anomalias", fmt_brl(float(valor_anom)))
    with col3:
        st.metric(
            "Descontos Anômalos",
            str(len(df_desc)),
            delta_color="inverse"
        )
    with col4:
        valor_desc = df_desc["desconto"].sum() if not df_desc.empty else 0
        st.metric("Valor em Descontos", fmt_brl(float(valor_desc)))

    st.divider()

    # ── Anomalias de vendas ──────────────────────────────────
    st.subheader("🚨 Vendas com Padrão Anômalo")
    st.caption("Detectadas por Isolation Forest — vendas com combinação incomum de valor, horário e desconto")

    if not df_anom.empty:
        df_anom_fmt = df_anom.copy()
        df_anom_fmt["data_venda"] = pd.to_datetime(
            df_anom_fmt["data_venda"]
        ).dt.strftime("%d/%m/%Y %H:%M")
        df_anom_fmt["total"]    = df_anom_fmt["total"].apply(fmt_brl)
        df_anom_fmt["desconto"] = df_anom_fmt["desconto"].apply(fmt_brl)
        df_anom_fmt["anomalia_score"] = df_anom_fmt["anomalia_score"].apply(
            lambda x: f"{x:.3f}"
        )

        df_anom_fmt.columns = [
            "Código", "Data", "Total",
            "Desconto", "Hora", "Score"
        ]

        st.dataframe(df_anom_fmt.head(20), use_container_width=True, hide_index=True)

        # Gráfico de distribuição por hora
        st.subheader("Distribuição por Hora do Dia")
        fig = px.histogram(
            df_anom,
            x="hora",
            nbins=24,
            labels={"hora": "Hora do Dia", "count": "Ocorrências"},
            color_discrete_sequence=["#B71C1C"],
        )
        fig.update_layout(
            plot_bgcolor="rgba(0,0,0,0)",
            paper_bgcolor="rgba(0,0,0,0)",
        )
        st.plotly_chart(fig, use_container_width=True)

    st.divider()

    # ── Descontos anômalos ───────────────────────────────────
    st.subheader("💸 Descontos Fora do Padrão")
    st.caption("Detectados por Z-score — descontos mais de 2 desvios padrão acima da média")

    if not df_desc.empty:
        df_desc_fmt = df_desc.copy()
        df_desc_fmt["data_venda"] = pd.to_datetime(
            df_desc_fmt["data_venda"]
        ).dt.strftime("%d/%m/%Y %H:%M")
        df_desc_fmt["total"]        = df_desc_fmt["total"].apply(fmt_brl)
        df_desc_fmt["desconto"]     = df_desc_fmt["desconto"].apply(fmt_brl)
        df_desc_fmt["pct_desconto"] = df_desc_fmt["pct_desconto"].apply(
            lambda x: f"{x:.1f}%"
        )
        df_desc_fmt["z_score"] = df_desc_fmt["z_score"].apply(
            lambda x: f"{x:.2f}"
        )

        df_desc_fmt = df_desc_fmt[[
            "venda_key", "data_venda", "vendedor_id",
            "total", "desconto", "pct_desconto", "z_score"
        ]]
        df_desc_fmt.columns = [
            "Código", "Data", "Vendedor",
            "Total", "Desconto", "% Desconto", "Z-Score"
        ]

        st.dataframe(df_desc_fmt.head(20), use_container_width=True, hide_index=True)

        # Descontos por vendedor
        if not df_desc.empty and "vendedor_id" in df_desc.columns:
            st.subheader("Descontos Anômalos por Vendedor")
            por_vendedor = df_desc.groupby("vendedor_id").agg(
                ocorrencias   = ("venda_key", "count"),
                desconto_total = ("desconto", "sum"),
            ).reset_index()

            fig_vend = px.bar(
                por_vendedor,
                x="vendedor_id",
                y="ocorrencias",
                color="vendedor_id",
                labels={
                    "vendedor_id": "Vendedor",
                    "ocorrencias": "Ocorrências"
                },
                color_discrete_sequence=px.colors.qualitative.Set2,
            )
            fig_vend.update_layout(
                plot_bgcolor="rgba(0,0,0,0)",
                paper_bgcolor="rgba(0,0,0,0)",
                showlegend=False,
            )
            st.plotly_chart(fig_vend, use_container_width=True)

    st.divider()

    # ── Performance de vendedores ────────────────────────────
    st.subheader("👤 Análise de Vendedores")
    st.caption("Comparativo de ticket médio e desconto entre vendedores")

    if not df_vend.empty:
        col1, col2 = st.columns(2)

        with col1:
            fig_ticket = px.bar(
                df_vend.sort_values("ticket_medio", ascending=True),
                x="ticket_medio",
                y="vendedor_id",
                orientation="h",
                labels={
                    "ticket_medio": "Ticket Médio (R$)",
                    "vendedor_id": "Vendedor"
                },
                color="z_ticket",
                color_continuous_scale="RdYlGn",
                color_continuous_midpoint=0,
            )
            fig_ticket.update_layout(
                plot_bgcolor="rgba(0,0,0,0)",
                paper_bgcolor="rgba(0,0,0,0)",
                title="Ticket Médio por Vendedor",
            )
            st.plotly_chart(fig_ticket, use_container_width=True)

        with col2:
            fig_desc = px.bar(
                df_vend.sort_values("desconto_medio", ascending=True),
                x="desconto_medio",
                y="vendedor_id",
                orientation="h",
                labels={
                    "desconto_medio": "Desconto Médio (%)",
                    "vendedor_id": "Vendedor"
                },
                color_discrete_sequence=["#1565C0"],
            )
            fig_desc.update_layout(
                plot_bgcolor="rgba(0,0,0,0)",
                paper_bgcolor="rgba(0,0,0,0)",
                title="Desconto Médio por Vendedor",
            )
            st.plotly_chart(fig_desc, use_container_width=True)

        st.subheader("Resumo por Vendedor")
        df_vend_fmt = df_vend.copy()
        df_vend_fmt["ticket_medio"]   = df_vend_fmt["ticket_medio"].apply(fmt_brl)
        df_vend_fmt["total_faturado"] = df_vend_fmt["total_faturado"].apply(fmt_brl)
        df_vend_fmt["desconto_medio"] = df_vend_fmt["desconto_medio"].apply(
            lambda x: f"{x:.2f}%"
        )
        df_vend_fmt["z_ticket"] = df_vend_fmt["z_ticket"].apply(
            lambda x: f"{x:.2f}"
        )

        df_vend_fmt.columns = [
            "Vendedor", "Total Vendas", "Ticket Médio",
            "Desconto Médio", "Total Faturado", "Z-Score Ticket"
        ]
        st.dataframe(df_vend_fmt, use_container_width=True, hide_index=True)

    st.divider()

    # ── Insight ──────────────────────────────────────────────
    st.subheader("💡 Insight")

    if not df_desc.empty and "vendedor_id" in df_desc.columns:
        vendedor_mais_anomalias = df_desc["vendedor_id"].value_counts().index[0]
        total_anomalias_vend   = df_desc["vendedor_id"].value_counts().iloc[0]
        st.warning(
            f"O vendedor **{vendedor_mais_anomalias}** concentra "
            f"**{total_anomalias_vend}** ocorrências de desconto anômalo. "
            f"Recomenda-se revisão das políticas de desconto com esse vendedor."
        )