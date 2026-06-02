import streamlit as st
import pandas as pd
from datetime import datetime
from core.db.postgres import read_query
from components.kpi_card import kpi_card

def render(client_key: str):
    st.title("🔆 Central de Comando")
    st.caption(f"Atualizado em {datetime.now().strftime('%d/%m/%Y %H:%M')}")

    try:
        # ── KPIs de vendas ───────────────────────────────────
        df_vendas = read_query(client_key, f"""
            SELECT 
                COUNT(*)                    AS total_vendas,
                COALESCE(SUM(total), 0)     AS faturamento,
                COALESCE(AVG(total), 0)     AS ticket_medio,
                COALESCE(SUM(desconto), 0)  AS total_desconto
            FROM client_{client_key}.vendas
            WHERE status = 'concluida'
        """)

        faturamento   = df_vendas["faturamento"].iloc[0]
        ticket_medio  = df_vendas["ticket_medio"].iloc[0]
        total_vendas  = df_vendas["total_vendas"].iloc[0]
        total_desconto = df_vendas["total_desconto"].iloc[0]

        # ── Cards de KPI ─────────────────────────────────────
        st.subheader("Visão Geral")
        col1, col2, col3, col4 = st.columns(4)

        with col1:
            kpi_card(
                "Faturamento Total",
                f"R$ {faturamento:,.2f}".replace(",", "X").replace(".", ",").replace("X", "."),
            )
        with col2:
            kpi_card(
                "Ticket Médio",
                f"R$ {ticket_medio:,.2f}".replace(",", "X").replace(".", ",").replace("X", "."),
            )
        with col3:
            kpi_card(
                "Total de Vendas",
                str(int(total_vendas)),
            )
        with col4:
            kpi_card(
                "Total de Descontos",
                f"R$ {total_desconto:,.2f}".replace(",", "X").replace(".", ",").replace("X", "."),
                delta_color="inverse"
            )

        st.divider()

        # ── Vendas por dia ───────────────────────────────────
        st.subheader("Faturamento por Dia")

        df_por_dia = read_query(client_key, f"""
            SELECT 
                DATE(data_venda)            AS dia,
                COUNT(*)                    AS vendas,
                COALESCE(SUM(total), 0)     AS faturamento
            FROM client_{client_key}.vendas
            WHERE status = 'concluida'
            GROUP BY DATE(data_venda)
            ORDER BY dia
        """)

        if not df_por_dia.empty:
            st.bar_chart(df_por_dia.set_index("dia")["faturamento"])

        st.divider()

        # ── Tabela de últimas vendas ─────────────────────────
        st.subheader("Últimas Vendas")

        df_ultimas = read_query(client_key, f"""
            SELECT 
                venda_key       AS "Código",
                TO_CHAR(data_venda, 'DD/MM/YYYY HH24:MI') AS "Data",
                vendedor_id     AS "Vendedor",
                total           AS "Total (R$)",
                desconto        AS "Desconto (R$)",
                status          AS "Status"
            FROM client_{client_key}.vendas
            ORDER BY data_venda DESC
            LIMIT 10
        """)

        st.dataframe(df_ultimas, use_container_width=True)

    except Exception as e:
        st.error(f"Erro ao carregar dados: {e}")
        st.info("Verifique se o pipeline de dados está rodando corretamente.")