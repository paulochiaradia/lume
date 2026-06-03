import sys
import os
sys.path.insert(0, os.path.dirname(__file__))

import streamlit as st
from core.access_control import get_allowed_pages, get_module

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

# ── Navegação por role ───────────────────────────────────────
role = st.session_state.get("role", "viewer")
name = st.session_state.get("name", "Usuário")

st.sidebar.title("🔆 Lume")
st.sidebar.caption(f"Olá, {name}")
st.sidebar.caption(f"Perfil: {role}")
st.sidebar.divider()

# Só mostra as páginas permitidas para o role
allowed_pages = get_allowed_pages(role)
selected = st.sidebar.selectbox("Navegação", allowed_pages)

st.sidebar.divider()
if st.sidebar.button("Sair"):
    st.session_state.clear()
    st.rerun()

st.sidebar.caption("Lume Inteligência Comercial")
st.sidebar.caption("v0.1.0 — MVP")

# ── Renderiza página ─────────────────────────────────────────
module = get_module(selected)

if module == "home":
    from pages.home import render
    render()
elif module == "produtos":
    from pages.produtos import render
    render()
else:
    st.title(f"{selected}")
    st.info("Este módulo será implementado nas próximas sprints.")