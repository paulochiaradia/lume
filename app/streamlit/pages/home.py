import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import streamlit as st
import pandas as pd
import plotly.express as px
from datetime import datetime
from core.api_client import (
    get_home_kpis,
    get_vendas_por_dia,
    get_estoque_alertas,
)

def fmt_brl(value: float) -> str:
    return f"R$ {value:,.2f}".replace(",", "X").replace(".", ",").replace("X", ".")

def get_insight_do_dia(kpis: dict) -> str:
    """Gera insight do dia via LLM"""
    try:
        import subprocess
        import json
        import os

        script = f"""
import sys
sys.path.insert(0, '/engine')
import os
os.environ['LLM_PROVIDER'] = '{os.getenv("LLM_PROVIDER", "mock")}'
from core.llm.client import get_llm_client
from core.llm.prompts.insight_diario import build_prompt
client = get_llm_client()
kpis = {json.dumps(kpis)}
prompt = build_prompt(kpis, 'estável')
print(client.complete(prompt))
"""
        result = subprocess.run(
            ["python", "-c", script],
            capture_output=True,
            text=True,
            timeout=30,
        )
        if result.returncode == 0:
            return result.stdout.strip()
        return "Análise em andamento..."
    except Exception:
        return "Análise em andamento..."

def render(client_key: str = None):
    st.title("🔆 Central de Comando")
    st.caption(f"Atualizado em {datetime.now().strftime('%d/%m/%Y %H:%M')}")

    with st.spinner("Carregando dados..."):
        kpis    = get_home_kpis()
        vendas  = get_vendas_por_dia()
        alertas = get_estoque_alertas()

    if not kpis:
        st.error("Erro ao carregar dados. Verifique a conexão com a API.")
        return

    # ── Alertas ativos ───────────────────────────────────────
    if alertas:
        st.warning(
            f"⚠️ **{len(alertas)} produto(s)** com estoque abaixo do mínimo. "
            f"Acesse o módulo de Estoque para detalhes."
        )

    # ── KPIs ─────────────────────────────────────────────────
    st.subheader("Visão Geral")
    col1, col2, col3, col4 = st.columns(4)

    with col1:
        st.metric("Faturamento Total", fmt_brl(kpis["faturamento"]))
    with col2:
        st.metric("Ticket Médio", fmt_brl(kpis["ticket_medio"]))
    with col3:
        st.metric("Total de Vendas", str(kpis["total_vendas"]))
    with col4:
        st.metric(
            "Total de Descontos",
            fmt_brl(kpis["total_desconto"]),
            delta_color="inverse"
        )

    st.divider()

    # ── Insight do dia ───────────────────────────────────────
    st.subheader("💡 Insight do Dia")
    with st.spinner("Gerando insight..."):
        insight = get_insight_do_dia(kpis)
    st.info(insight)

    st.divider()

    # ── Gráfico de faturamento ───────────────────────────────
    st.subheader("Faturamento por Dia")

    if vendas:
        df = pd.DataFrame(vendas)

        fig = px.bar(
            df,
            x="dia",
            y="faturamento",
            labels={"dia": "Data", "faturamento": "Faturamento (R$)"},
            color_discrete_sequence=["#1B5E20"],
        )
        fig.update_layout(
            plot_bgcolor="rgba(0,0,0,0)",
            paper_bgcolor="rgba(0,0,0,0)",
        )
        st.plotly_chart(fig, use_container_width=True)

    st.divider()

    # ── Resumo por dia ───────────────────────────────────────
    st.subheader("Resumo por Dia")

    if vendas:
        df_tabela = pd.DataFrame(vendas)
        df_tabela["faturamento"] = df_tabela["faturamento"].apply(fmt_brl)
        df_tabela.columns        = ["Data", "Vendas", "Faturamento"]
        st.dataframe(df_tabela, use_container_width=True, hide_index=True)