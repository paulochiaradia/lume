import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import streamlit as st
from core.api_client import login

def render():
    st.title("🔆 Lume Inteligência Comercial")
    st.subheader("Acesse sua conta")

    with st.form("login_form"):
        email = st.text_input("E-mail")
        password = st.text_input("Senha", type="password")
        submitted = st.form_submit_button("Entrar", use_container_width=True)

    if submitted:
        if not email or not password:
            st.error("Preencha e-mail e senha.")
            return

        with st.spinner("Autenticando..."):
            result = login(email, password)

        if result:
            st.session_state["token"]     = result["token"]
            st.session_state["role"]      = result["role"]
            st.session_state["client_key"] = result["client_key"]
            st.session_state["name"]      = result["name"]
            st.session_state["logged_in"] = True
            st.rerun()
        else:
            st.error("E-mail ou senha incorretos.")