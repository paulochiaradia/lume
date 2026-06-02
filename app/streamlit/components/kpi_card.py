import streamlit as st

def kpi_card(label: str, value: str, delta: str = None, delta_color: str = "normal"):
    """
    Renderiza um card de KPI padronizado
    
    Args:
        label: Nome do indicador
        value: Valor principal formatado
        delta: Variação em relação ao período anterior
        delta_color: "normal", "inverse" ou "off"
    """
    st.metric(
        label=label,
        value=value,
        delta=delta,
        delta_color=delta_color
    )