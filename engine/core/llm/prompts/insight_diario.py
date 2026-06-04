def build_prompt(kpis: dict, tendencia: str) -> str:
    return f"""
Analise os dados de vendas abaixo e gere um insight acionável em 2-3 frases
para o gestor de uma loja de materiais de construção.

DADOS DO DIA:
- Faturamento: R$ {kpis.get('faturamento', 0):,.2f}
- Ticket médio: R$ {kpis.get('ticket_medio', 0):,.2f}
- Total de vendas: {kpis.get('total_vendas', 0)}
- Tendência vs semana passada: {tendencia}

Foque em ações concretas que o gestor pode tomar hoje.
Responda em português brasileiro, de forma direta e objetiva.
""".strip()