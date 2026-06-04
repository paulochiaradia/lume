import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import streamlit as st
import pandas as pd
import plotly.express as px
from core.api_client import get_clientes_rfm, get_resumo_segmentos

CORES_SEGMENTO = {
    "campeon":    "#1B5E20",   # verde escuro
    "fiel":       "#4CAF50",   # verde médio
    "promissor":  "#1565C0",   # azul escuro
    "novo":       "#42A5F5",   # azul claro
    "em_risco":   "#FF6F00",   # laranja escuro
    "hibernando": "#FFD54F",   # amarelo
    "perdido":    "#B71C1C",   # vermelho escuro
    "outros":     "#90A4AE",   # cinza azulado
}

LABELS_SEGMENTO = {
    "campeon":    "Campeão",
    "fiel":       "Fiel",
    "promissor":  "Promissor",
    "novo":       "Novo",
    "em_risco":   "Em Risco",
    "hibernando": "Hibernando",
    "perdido":    "Perdido",
    "outros":     "Outros",
}

def fmt_brl(value: float) -> str:
    return f"R$ {value:,.2f}".replace(",", "X").replace(".", ",").replace("X", ".")

def render():
    st.title("👥 Clientes")
    st.caption("Segmentação RFM — Recência, Frequência e Valor Monetário")

    with st.spinner("Carregando dados de clientes..."):
        segmentos = get_resumo_segmentos()
        clientes  = get_clientes_rfm()

    if not segmentos:
        st.warning("Nenhum dado de clientes encontrado.")
        return

    df_seg = pd.DataFrame(segmentos)
    df_cli = pd.DataFrame(clientes) if clientes else pd.DataFrame()

    # ── KPIs ─────────────────────────────────────────────────
    st.subheader("Visão Geral")

    total_clientes = df_seg["clientes"].sum()
    valor_total    = df_seg["valor_total"].sum()
    em_risco       = df_seg[df_seg["segmento"].isin(["em_risco", "hibernando"])]["clientes"].sum()
    campeoes       = df_seg[df_seg["segmento"] == "campeon"]["clientes"].sum()

    col1, col2, col3, col4 = st.columns(4)
    with col1:
        st.metric("Total de Clientes", str(int(total_clientes)))
    with col2:
        st.metric("Faturamento Total", fmt_brl(float(valor_total)))
    with col3:
        st.metric("Campeões", str(int(campeoes)))
    with col4:
        st.metric("Em Risco", str(int(em_risco)), delta_color="inverse")

    st.divider()

    # ── Gráfico de pizza — distribuição por segmento ─────────
    col1, col2 = st.columns(2)

    with col1:
        st.subheader("Distribuição por Segmento")

        df_seg["label"] = df_seg["segmento"].map(LABELS_SEGMENTO)
        df_seg["cor"]   = df_seg["segmento"].map(CORES_SEGMENTO)

        fig_pizza = px.pie(
            df_seg,
            values="clientes",
            names="label",
            color="segmento",
            color_discrete_map=CORES_SEGMENTO,
            hole=0.4,
        )
        fig_pizza.update_layout(
            plot_bgcolor="rgba(0,0,0,0)",
            paper_bgcolor="rgba(0,0,0,0)",
        )
        st.plotly_chart(fig_pizza, use_container_width=True)

    with col2:
        st.subheader("Faturamento por Segmento")

        fig_bar = px.bar(
            df_seg.sort_values("valor_total"),
            x="valor_total",
            y="label",
            orientation="h",
            color="segmento",
            color_discrete_map=CORES_SEGMENTO,
            labels={"valor_total": "Faturamento (R$)", "label": "Segmento"},
        )
        fig_bar.update_layout(
            plot_bgcolor="rgba(0,0,0,0)",
            paper_bgcolor="rgba(0,0,0,0)",
            showlegend=False,
        )
        st.plotly_chart(fig_bar, use_container_width=True)

    st.divider()

    # ── Tabela de segmentos ──────────────────────────────────
    st.subheader("Detalhamento por Segmento")

    df_tabela = df_seg[[
        "label", "clientes", "valor_total",
        "valor_medio", "recencia_media"
    ]].copy()

    df_tabela["valor_total"]   = df_tabela["valor_total"].apply(fmt_brl)
    df_tabela["valor_medio"]   = df_tabela["valor_medio"].apply(fmt_brl)
    df_tabela["recencia_media"] = df_tabela["recencia_media"].apply(
        lambda x: f"{int(x)} dias"
    )

    df_tabela.columns = [
        "Segmento", "Clientes", "Faturamento Total",
        "Ticket Médio", "Recência Média"
    ]

    st.dataframe(df_tabela, use_container_width=True, hide_index=True)

    st.divider()

    # ── Clientes em risco ────────────────────────────────────
    st.subheader("🚨 Clientes em Risco — Ação Necessária")

    if not df_cli.empty:
        df_risco = df_cli[df_cli["recencia"] > 45].sort_values(
            "valor_total", ascending=False
        ).head(10)

        if not df_risco.empty:
            df_risco_tabela = df_risco[[
                "cliente_id", "recencia",
                "frequencia", "valor_total"
            ]].copy()

            df_risco_tabela["valor_total"] = df_risco_tabela["valor_total"].apply(fmt_brl)
            df_risco_tabela.columns = [
                "Cliente", "Dias sem comprar",
                "Total de Compras", "Valor Histórico"
            ]

            st.dataframe(df_risco_tabela, use_container_width=True, hide_index=True)

            st.warning(
                f"{len(df_risco)} clientes estratégicos sem compra há mais de 45 dias. "
                f"Entre em contato esta semana."
            )
        else:
            st.success("Nenhum cliente estratégico em risco no momento.")

    st.divider()

    # ── Insight LLM ──────────────────────────────────────────
    st.subheader("💡 Insight")
    st.info(
        f"Você tem **{int(campeoes)} clientes campeões** que representam "
        f"a maior parte do faturamento. "
        f"Mantenha relacionamento próximo com esses clientes. "
        f"**{int(em_risco)} clientes** precisam de atenção imediata."
    )