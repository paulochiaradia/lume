import streamlit as st

st.set_page_config(
    page_title="Lume Inteligência Comercial",
    page_icon="🔆",
    layout="wide"
)

CLIENT_KEY = "loja_teste"

st.sidebar.title("Navegação")

pages = {
    "🏠 Home": "home",
    "📦 Estoque": "estoque",
    "👥 Clientes": "clientes",
    "📊 Vendas": "vendas",
}

selected = st.sidebar.selectbox("Ir para", list(pages.keys()))

st.sidebar.divider()
st.sidebar.caption("Lume Inteligência Comercial")
st.sidebar.caption("v0.1.0 — MVP")

page = pages[selected]

if page == "home":
    from pages.home import render
    render(CLIENT_KEY)
else:
    st.title("Módulo em desenvolvimento")
    st.info("Este módulo será implementado nas próximas sprints.")