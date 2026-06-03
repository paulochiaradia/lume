import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import streamlit as st
import pandas as pd
from datetime import datetime
from core.api_client import get_home_kpis, get_vendas_por_dia
from components.kpi_card import kpi_card

def fmt_brl(value: float) -> str:
    return f"R$ {value:,.2f}".replace(",", "X").replace(".", ",").replace("X", ".")

def render(client_key: str = None):
    st.title("🔆 Central de Comando")
    st.caption(f"Atualizado em {datetime.now().strftime('%d/%m/%Y %H:%M')}")

    # ── KPIs ─────────────────────────────────────────────────
    kpis = get_home_kpis()

    if not kpis:
        st.error("Erro ao carregar KPIs. Verifique a conexão com a API.")
        return

    st.subheader("Visão Geral")
    col1, col2, col3, col4 = st.columns(4)

    with col1:
        kpi_card("Faturamento Total", fmt_brl(kpis["faturamento"]))
    with col2:
        kpi_card("Ticket Médio", fmt_brl(kpis["ticket_medio"]))
    with col3:
        kpi_card("Total de Vendas", str(kpis["total_vendas"]))
    with col4:
        kpi_card("Total de Descontos", fmt_brl(kpis["total_desconto"]), delta_color="inverse")

    st.divider()

    # ── Vendas por dia ───────────────────────────────────────
    st.subheader("Faturamento por Dia")

    vendas = get_vendas_por_dia()
    if vendas:
        df = pd.DataFrame(vendas)
        df = df.set_index("dia")
        st.bar_chart(df["faturamento"])

    st.divider()

    # ── Tabela de vendas por dia ─────────────────────────────
    st.subheader("Resumo por Dia")
    if vendas:
        df_tabela = pd.DataFrame(vendas)
        df_tabela.columns = ["Data", "Vendas", "Faturamento (R$)"]
        st.dataframe(df_tabela, use_container_width=True)