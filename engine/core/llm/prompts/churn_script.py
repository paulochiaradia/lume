def build_prompt(clientes_em_risco: list) -> str:
    clientes_str = "\n".join([
        f"- Cliente {c['cliente_id']}: "
        f"último contato há {c['recencia']} dias, "
        f"valor histórico R$ {c['valor_total']:,.2f}"
        for c in clientes_em_risco[:5]
    ])

    return f"""
Gere um script de abordagem para um vendedor entrar em contato
com clientes que estão sumindo de uma loja de materiais de construção.

CLIENTES EM RISCO:
{clientes_str}

O script deve:
1. Ser natural e não soar como telemarketing
2. Mencionar que a loja sentiu falta do cliente
3. Perguntar se há obra em andamento ou planejada
4. Oferecer ajuda para orçamento

Máximo 5 linhas. Em português brasileiro.
""".strip()