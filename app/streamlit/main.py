import sys
import os
sys.path.insert(0, os.path.dirname(__file__))

import streamlit as st

st.set_page_config(
    page_title="Lume Inteligência Comercial",
    page_icon="🔆",
    layout="wide"
)

# ── Autenticação ─────────────────────────────────────────────
if not st.session_state.get("logged_in"):
    from pages.login import render
    render()
    st.stop()

# ── Navegação ────────────────────────────────────────────────
role = st.session_state.get("role", "viewer")
name = st.session_state.get("name", "Usuário")

st.sidebar.title("🔆 Lume")
st.sidebar.caption(f"Olá, {name}")
st.sidebar.caption(f"Perfil: {role}")
st.sidebar.divider()

pages = {
    "🏠 Home":      "home",
    "📊 Vendas":    "vendas",
    "📦 Estoque":   "estoque",
    "👥 Clientes":  "clientes",
    "🛒 Produtos":  "produtos",
}

selected = st.sidebar.selectbox("Navegação", list(pages.keys()))

st.sidebar.divider()
if st.sidebar.button("Sair"):
    st.session_state.clear()
    st.rerun()

st.sidebar.caption("Lume Inteligência Comercial")
st.sidebar.caption("v0.1.0 — MVP")

# ── Renderiza página ─────────────────────────────────────────
page = pages[selected]

if page == "home":
    from pages.home import render
    render()
else:
    st.title("Módulo em desenvolvimento")
    st.info("Este módulo será implementado nas próximas sprints.")