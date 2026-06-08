import os
import logging
from datetime import datetime
import duckdb
import pandas as pd

log = logging.getLogger(__name__)

TEMPLATE_DIR = os.path.join(os.path.dirname(__file__), "templates")
DUCKDB_PATH  = os.getenv("DUCKDB_PATH", "/app/data/duckdb")
REPORTS_DIR  = os.getenv("REPORTS_DIR", "/app/data/reports")


def fmt_brl(value: float) -> str:
    return f"R$ {value:,.2f}".replace(",", "X").replace(".", ",").replace("X", ".")


def load_duckdb(client_key: str, table: str) -> pd.DataFrame:
    path = os.path.join(DUCKDB_PATH, f"{client_key}.duckdb")
    if not os.path.exists(path):
        return pd.DataFrame()
    try:
        conn = duckdb.connect(path)
        df   = conn.execute(f"SELECT * FROM {table}").df()
        conn.close()
        return df
    except Exception:
        return pd.DataFrame()


def _gerar_pdf_reportlab(caminho_pdf: str, dados: dict):
    """Gera PDF usando ReportLab"""
    from reportlab.lib.pagesizes import A4
    from reportlab.lib.styles import getSampleStyleSheet, ParagraphStyle
    from reportlab.lib.units import cm
    from reportlab.lib import colors
    from reportlab.platypus import SimpleDocTemplate, Paragraph, Spacer, Table, TableStyle
    from reportlab.lib.enums import TA_CENTER

    doc    = SimpleDocTemplate(caminho_pdf, pagesize=A4,
                               leftMargin=2*cm, rightMargin=2*cm,
                               topMargin=2*cm, bottomMargin=2*cm)
    styles = getSampleStyleSheet()

    VERDE   = colors.HexColor("#1B5E20")
    CINZA   = colors.HexColor("#f5f5f5")
    LARANJA = colors.HexColor("#E65100")

    titulo_style = ParagraphStyle(
        "titulo", parent=styles["Heading1"],
        textColor=VERDE, fontSize=18, spaceAfter=6
    )
    subtitulo_style = ParagraphStyle(
        "subtitulo", parent=styles["Heading2"],
        textColor=VERDE, fontSize=12, spaceAfter=6
    )
    normal_style = styles["Normal"]
    normal_style.fontSize = 10

    story = []

    # Cabeçalho
    story.append(Paragraph("Lume Inteligência Comercial", titulo_style))
    story.append(Paragraph(
        f"Relatório Executivo Semanal — {dados['cliente_nome']}", subtitulo_style
    ))
    story.append(Paragraph(f"Gerado em {dados['data_geracao']}", normal_style))
    story.append(Spacer(1, 0.5*cm))

    # KPIs
    story.append(Paragraph("Performance Geral", subtitulo_style))
    kpi_data = [
        ["Faturamento Total", "Ticket Médio", "Total Vendas", "Total Descontos"],
        [dados["faturamento"], dados["ticket_medio"],
         dados["total_vendas"], dados["total_descontos"]],
    ]
    kpi_table = Table(kpi_data, colWidths=[4.25*cm]*4)
    kpi_table.setStyle(TableStyle([
        ("BACKGROUND", (0, 0), (-1, 0), VERDE),
        ("TEXTCOLOR",  (0, 0), (-1, 0), colors.white),
        ("BACKGROUND", (0, 1), (-1, 1), CINZA),
        ("ALIGN",      (0, 0), (-1, -1), "CENTER"),
        ("FONTSIZE",   (0, 0), (-1, -1), 9),
        ("GRID",       (0, 0), (-1, -1), 0.5, colors.white),
        ("BOX",        (0, 0), (-1, -1), 1, VERDE),
    ]))
    story.append(kpi_table)
    story.append(Spacer(1, 0.5*cm))

    # Insight
    story.append(Paragraph("Insight do Período", subtitulo_style))
    story.append(Paragraph(dados["insight_texto"], normal_style))
    story.append(Spacer(1, 0.5*cm))

    # Top produtos
    story.append(Paragraph("Top Produtos — Curva ABC", subtitulo_style))
    if dados["top_produtos"]:
        prod_data = [["Produto", "Categoria", "Faturamento", "Classe"]]
        for p in dados["top_produtos"]:
            prod_data.append([p["nome"], p["categoria"], p["faturamento"], p["classe"]])
        prod_table = Table(prod_data, colWidths=[6*cm, 3.5*cm, 3.5*cm, 2*cm])
        prod_table.setStyle(TableStyle([
            ("BACKGROUND",     (0, 0), (-1, 0), VERDE),
            ("TEXTCOLOR",      (0, 0), (-1, 0), colors.white),
            ("FONTSIZE",       (0, 0), (-1, -1), 9),
            ("GRID",           (0, 0), (-1, -1), 0.3, colors.lightgrey),
            ("ROWBACKGROUNDS", (0, 1), (-1, -1), [colors.white, CINZA]),
        ]))
        story.append(prod_table)
    story.append(Spacer(1, 0.5*cm))

    # Clientes em risco
    story.append(Paragraph("Clientes em Risco de Churn", subtitulo_style))
    if dados["clientes_risco"]:
        risco_data = [["Cliente", "Dias sem comprar", "Valor Histórico"]]
        for c in dados["clientes_risco"]:
            risco_data.append([
                c["cliente_id"],
                f"{c['recencia']} dias",
                c["valor_total"]
            ])
        risco_table = Table(risco_data, colWidths=[6*cm, 4*cm, 5*cm])
        risco_table.setStyle(TableStyle([
            ("BACKGROUND",     (0, 0), (-1, 0), LARANJA),
            ("TEXTCOLOR",      (0, 0), (-1, 0), colors.white),
            ("FONTSIZE",       (0, 0), (-1, -1), 9),
            ("GRID",           (0, 0), (-1, -1), 0.3, colors.lightgrey),
            ("ROWBACKGROUNDS", (0, 1), (-1, -1), [colors.white, CINZA]),
        ]))
        story.append(risco_table)
    else:
        story.append(Paragraph("Nenhum cliente estratégico em risco.", normal_style))
    story.append(Spacer(1, 0.5*cm))

    # Alertas de estoque
    story.append(Paragraph("Alertas de Estoque", subtitulo_style))
    if dados["alertas_estoque"]:
        est_data = [["Produto", "Qtd Atual", "Qtd Mínima"]]
        for a in dados["alertas_estoque"]:
            est_data.append([a["nome"], a["quantidade"], a["quantidade_min"]])
        est_table = Table(est_data, colWidths=[8*cm, 3*cm, 4*cm])
        est_table.setStyle(TableStyle([
            ("BACKGROUND",     (0, 0), (-1, 0), LARANJA),
            ("TEXTCOLOR",      (0, 0), (-1, 0), colors.white),
            ("FONTSIZE",       (0, 0), (-1, -1), 9),
            ("GRID",           (0, 0), (-1, -1), 0.3, colors.lightgrey),
            ("ROWBACKGROUNDS", (0, 1), (-1, -1), [colors.white, CINZA]),
        ]))
        story.append(est_table)
    else:
        story.append(Paragraph("Estoque em níveis adequados.", normal_style))

    # Rodapé
    story.append(Spacer(1, 1*cm))
    rodape_style = ParagraphStyle(
        "rodape", parent=styles["Normal"],
        fontSize=8, textColor=colors.grey, alignment=TA_CENTER
    )
    story.append(Paragraph(
        f"Lume Inteligência Comercial — gerado automaticamente em {dados['data_geracao']}",
        rodape_style
    ))

    doc.build(story)


def gerar_relatorio_semanal(
    client_key: str,
    cliente_nome: str,
    kpis: dict,
) -> str:
    """
    Gera o relatório semanal em PDF.
    Retorna o caminho do arquivo gerado.
    """
    os.makedirs(REPORTS_DIR, exist_ok=True)

    # ── Carrega dados do DuckDB ──────────────────────────────
    df_abc = load_duckdb(client_key, "abc_resultado")
    df_rfm = load_duckdb(client_key, "rfm_resultado")

    # Top produtos ABC
    top_produtos = []
    if not df_abc.empty:
        for _, row in df_abc.head(10).iterrows():
            top_produtos.append({
                "nome":        row.get("produto_key", ""),
                "categoria":   row.get("categoria", ""),
                "faturamento": fmt_brl(float(row.get("faturamento", 0))),
                "classe":      row.get("classe", "C"),
            })

    # Clientes em risco
    clientes_risco = []
    if not df_rfm.empty:
        em_risco = df_rfm[
            df_rfm["segmento"].isin(["em_risco", "hibernando"])
        ].sort_values("valor_total", ascending=False).head(5)

        for _, row in em_risco.iterrows():
            clientes_risco.append({
                "cliente_id":  row.get("cliente_id", ""),
                "recencia":    int(row.get("recencia", 0)),
                "valor_total": fmt_brl(float(row.get("valor_total", 0))),
            })

    # Alertas de estoque
    alertas_estoque = []
    try:
        from core.db.postgres import read_query
        df_est = read_query(client_key, f"""
            SELECT
                e.produto_key,
                COALESCE(p.nome, e.produto_key) AS nome,
                e.quantidade,
                e.quantidade_min
            FROM client_{client_key}.estoque e
            LEFT JOIN client_{client_key}.produtos p
                ON p.produto_key = e.produto_key
            WHERE e.quantidade <= e.quantidade_min
            ORDER BY e.quantidade ASC
            LIMIT 10
        """)
        for _, row in df_est.iterrows():
            alertas_estoque.append({
                "nome":           row["nome"],
                "quantidade":     f"{row['quantidade']:.0f}",
                "quantidade_min": f"{row['quantidade_min']:.0f}",
            })
    except Exception as e:
        log.warning(f"Erro ao buscar alertas de estoque: {e}")

    # Insight via LLM
    from core.llm.client import get_llm_client
    llm = get_llm_client()
    insight_texto = llm.complete(
        f"relatorio semanal para loja de materiais de construção. "
        f"Faturamento: {kpis.get('faturamento', 0):.2f}. "
        f"Total vendas: {kpis.get('total_vendas', 0)}. "
        f"Clientes em risco: {len(clientes_risco)}. "
        f"Alertas de estoque: {len(alertas_estoque)}."
    )

    # Monta dados para o PDF
    dados_pdf = {
        "cliente_nome":    cliente_nome,
        "data_geracao":    datetime.now().strftime("%d/%m/%Y %H:%M"),
        "faturamento":     fmt_brl(float(kpis.get("faturamento", 0))),
        "ticket_medio":    fmt_brl(float(kpis.get("ticket_medio", 0))),
        "total_vendas":    str(int(kpis.get("total_vendas", 0))),
        "total_descontos": fmt_brl(float(kpis.get("total_desconto", 0))),
        "insight_texto":   insight_texto,
        "top_produtos":    top_produtos,
        "clientes_risco":  clientes_risco,
        "alertas_estoque": alertas_estoque,
    }

    # Gera PDF
    nome_arquivo = f"relatorio_{client_key}_{datetime.now().strftime('%Y%m%d_%H%M%S')}.pdf"
    caminho_pdf  = os.path.join(REPORTS_DIR, nome_arquivo)

    try:
        _gerar_pdf_reportlab(caminho_pdf, dados_pdf)
        log.info(f"Relatorio gerado: {caminho_pdf}")
        return caminho_pdf
    except Exception as e:
        log.error(f"Erro ao gerar PDF: {e}")
        return None