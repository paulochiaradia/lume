import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import streamlit as st
import requests
from core.api_client import get_home_kpis

def render():
    st.title("📄 Relatórios")
    st.caption("Geração automática de relatórios executivos em PDF")

    client_key   = st.session_state.get("client_key", "loja_teste")
    cliente_nome = st.session_state.get("name", "Cliente")

    # ── Relatório Semanal ────────────────────────────────────
    st.subheader("Relatório Executivo Semanal")
    st.write(
        "Gera um PDF completo com KPIs, curva ABC, "
        "clientes em risco e alertas de estoque."
    )

    if st.button("📥 Gerar Relatório Agora", use_container_width=True):
        with st.spinner("Gerando relatório... aguarde."):
            try:
                kpis = get_home_kpis()
                if not kpis:
                    st.error("Erro ao buscar KPIs.")
                    return

                import subprocess
                import json

                script = f"""
import sys
sys.path.insert(0, '/engine')
import os
os.environ['POSTGRES_DSN'] = '{os.getenv("POSTGRES_DSN", "")}'
os.environ['DUCKDB_PATH'] = '{os.getenv("DUCKDB_PATH", "/app/data/duckdb")}'
os.environ['REPORTS_DIR'] = '{os.getenv("REPORTS_DIR", "/app/data/reports")}'
os.environ['LLM_PROVIDER'] = '{os.getenv("LLM_PROVIDER", "mock")}'
from core.reports.generator import gerar_relatorio_semanal
kpis = {json.dumps(kpis)}
caminho = gerar_relatorio_semanal('{client_key}', '{cliente_nome}', kpis)
print(caminho)
"""
                result = subprocess.run(
                    ["python", "-c", script],
                    capture_output=True,
                    text=True,
                    timeout=120,
                )

                if result.returncode == 0:
                    caminho = result.stdout.strip().split("\n")[-1]
                    if caminho and os.path.exists(caminho):
                        with open(caminho, "rb") as f:
                            pdf_bytes = f.read()
                        st.success("Relatório gerado com sucesso!")
                        st.download_button(
                            label="⬇️ Baixar PDF",
                            data=pdf_bytes,
                            file_name=f"relatorio_{client_key}.pdf",
                            mime="application/pdf",
                            use_container_width=True,
                        )
                    else:
                        st.error(f"Arquivo não encontrado: {caminho}")
                        st.code(result.stdout)
                else:
                    st.error("Erro ao gerar relatório.")
                    st.code(result.stderr)

            except Exception as e:
                st.error(f"Erro: {e}")
                import traceback
                st.code(traceback.format_exc())

    st.divider()

    # ── Histórico de relatórios ──────────────────────────────
    st.subheader("Histórico de Relatórios")

    reports_dir = os.getenv("REPORTS_DIR", "/app/data/reports")
    if os.path.exists(reports_dir):
        arquivos = sorted(
            [f for f in os.listdir(reports_dir)
             if f.endswith(".pdf") and client_key in f],
            reverse=True
        )

        if arquivos:
            for arquivo in arquivos[:10]:
                col1, col2 = st.columns([4, 1])
                with col1:
                    st.text(arquivo)
                with col2:
                    caminho = os.path.join(reports_dir, arquivo)
                    with open(caminho, "rb") as f:
                        st.download_button(
                            label="⬇️",
                            data=f.read(),
                            file_name=arquivo,
                            mime="application/pdf",
                            key=arquivo,
                        )
        else:
            st.info("Nenhum relatório gerado ainda.")
    else:
        st.info("Nenhum relatório gerado ainda.")

    st.divider()

    # ── Insight ──────────────────────────────────────────────
    st.subheader("💡 Sobre os Relatórios")
    st.info(
        "O relatório semanal é gerado automaticamente toda segunda-feira "
        "e enviado por e-mail para o gestor. "
        "Você também pode gerar um relatório a qualquer momento clicando no botão acima."
    )