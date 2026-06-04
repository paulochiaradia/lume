import os
import requests
import streamlit as st

API_URL = os.getenv("API_URL", "http://collector:8080")

def get_token() -> str:
    """Retorna o token JWT da sessão atual"""
    return st.session_state.get("token", "")

def get_headers() -> dict:
    """Retorna os headers com o token JWT"""
    return {
        "Authorization": f"Bearer {get_token()}",
        "Content-Type": "application/json",
    }

def login(email: str, password: str) -> dict:
    """Autentica o usuário e retorna os dados da sessão"""
    response = requests.post(
        f"{API_URL}/api/v1/auth/login",
        json={"email": email, "password": password},
        timeout=10,
    )
    if response.status_code == 200:
        return response.json()
    return None

def get_home_kpis() -> dict:
    """Busca os KPIs da home"""
    response = requests.get(
        f"{API_URL}/api/v1/home/kpis",
        headers=get_headers(),
        timeout=10,
    )
    if response.status_code == 200:
        return response.json()
    return None

def get_vendas_por_dia() -> list:
    """Busca as vendas por dia"""
    response = requests.get(
        f"{API_URL}/api/v1/vendas/por-dia",
        headers=get_headers(),
        timeout=10,
    )
    if response.status_code == 200:
        return response.json()
    return []

def get_produtos_abc() -> list:
    """Busca a curva ABC dos produtos"""
    response = requests.get(
        f"{API_URL}/api/v1/produtos/abc",
        headers=get_headers(),
        timeout=10,
    )
    if response.status_code == 200:
        return response.json()
    return []

def get_estoque_alertas() -> list:
    """Busca os alertas de estoque"""
    response = requests.get(
        f"{API_URL}/api/v1/estoque/alertas",
        headers=get_headers(),
        timeout=10,
    )
    if response.status_code == 200:
        return response.json()
    return []

def get_clientes_rfm() -> list:
    """Busca clientes com dados RFM"""
    response = requests.get(
        f"{API_URL}/api/v1/clientes/rfm",
        headers=get_headers(),
        timeout=10,
    )
    if response.status_code == 200:
        return response.json()
    return []

def get_resumo_segmentos() -> list:
    """Busca resumo de segmentos de clientes"""
    response = requests.get(
        f"{API_URL}/api/v1/clientes/segmentos",
        headers=get_headers(),
        timeout=10,
    )
    if response.status_code == 200:
        return response.json()
    return []