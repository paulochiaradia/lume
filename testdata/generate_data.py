"""
Lume — Gerador de dados sintéticos
Gera 2 anos de histórico realista de uma loja de materiais de construção
"""

import csv
import random
import os
from datetime import datetime, timedelta

random.seed(42)

# ── Configurações ────────────────────────────────────────────
OUTPUT_DIR = os.path.dirname(os.path.abspath(__file__))
START_DATE = datetime(2023, 1, 1)
END_DATE   = datetime(2024, 12, 31)

# ── Produtos realistas ───────────────────────────────────────
PRODUTOS = [
    # (codigo, nome, categoria, subcategoria, unidade, custo, preco)
    ("P001", "Cimento CP-II 50kg",        "Argamassas",    "Cimento",    "SC",  28.00,  35.00),
    ("P002", "Cimento CP-V 50kg",         "Argamassas",    "Cimento",    "SC",  30.00,  38.00),
    ("P003", "Areia Media 20kg",          "Agregados",     "Areia",      "SC",   8.00,  12.00),
    ("P004", "Areia Fina 20kg",           "Agregados",     "Areia",      "SC",   7.50,  11.00),
    ("P005", "Brita 1 20kg",              "Agregados",     "Brita",      "SC",  10.00,  15.00),
    ("P006", "Tijolo Ceramico 9cm",       "Alvenaria",     "Tijolos",    "UN",   0.85,   1.20),
    ("P007", "Tijolo Ceramico 6cm",       "Alvenaria",     "Tijolos",    "UN",   0.70,   1.00),
    ("P008", "Bloco de Concreto",         "Alvenaria",     "Blocos",     "UN",   2.50,   3.80),
    ("P009", "Ferro 10mm 12m",            "Estrutural",    "Ferro",      "BR",  42.00,  55.00),
    ("P010", "Ferro 8mm 12m",             "Estrutural",    "Ferro",      "BR",  28.00,  38.00),
    ("P011", "Ferro 12mm 12m",            "Estrutural",    "Ferro",      "BR",  58.00,  75.00),
    ("P012", "Telha Ceramica",            "Cobertura",     "Telhas",     "UN",   2.50,   3.80),
    ("P013", "Telha Fibrocimento 2.44m",  "Cobertura",     "Telhas",     "UN",  18.00,  25.00),
    ("P014", "Tinta Latex 18L",           "Acabamento",    "Tintas",     "GL",  65.00,  89.00),
    ("P015", "Tinta Acrilica 18L",        "Acabamento",    "Tintas",     "GL",  72.00,  98.00),
    ("P016", "Rejunte 5kg",               "Acabamento",    "Rejunte",    "PCT",  8.50,  13.00),
    ("P017", "Massa Corrida 25kg",        "Acabamento",    "Massa",      "BD",  28.00,  38.00),
    ("P018", "Disco de Corte 4.5",        "Ferramentas",   "Discos",     "UN",   4.50,   8.00),
    ("P019", "Rebolo Desbaste 4.5",       "Ferramentas",   "Rebolos",    "UN",   6.00,  10.50),
    ("P020", "Broca de Concreto 10mm",    "Ferramentas",   "Brocas",     "UN",   8.00,  14.00),
    ("P021", "Mangueira 1/2 50m",         "Hidraulica",    "Mangueiras", "RL",  45.00,  65.00),
    ("P022", "Tubo PVC 100mm 6m",         "Hidraulica",    "Tubos",      "BR",  28.00,  42.00),
    ("P023", "Tubo PVC 50mm 6m",          "Hidraulica",    "Tubos",      "BR",  18.00,  28.00),
    ("P024", "Joelho PVC 100mm",          "Hidraulica",    "Conexoes",   "UN",   4.50,   7.00),
    ("P025", "Fio Eletrico 2.5mm 100m",   "Eletrica",      "Fios",       "RL", 180.00, 240.00),
    ("P026", "Disjuntor 20A",             "Eletrica",      "Disjuntores","UN",  18.00,  28.00),
    ("P027", "Tomada 2P+T",               "Eletrica",      "Tomadas",    "UN",   8.00,  14.00),
    ("P028", "Impermeabilizante 18L",     "Impermeab.",    "Liquidos",   "GL",  85.00, 120.00),
    ("P029", "Argamassa AC-II 20kg",      "Argamassas",    "Argamassa",  "SC",  18.00,  26.00),
    ("P030", "Gesso 20kg",                "Acabamento",    "Gesso",      "SC",  12.00,  18.00),
]

# ── Clientes ─────────────────────────────────────────────────
TIPOS_CLIENTE = [
    ("construtora",    0.10),  # 10% — compram muito, ticket alto
    ("autonomo",       0.35),  # 35% — frequência média
    ("pessoa_fisica",  0.55),  # 55% — compras esporádicas
]

BAIRROS = [
    "Centro", "Vila Nova", "Jardim América", "Parque Industrial",
    "Vila Operária", "Residencial Sul", "Bairro Alto", "Santa Cruz"
]

def gerar_clientes(n=150):
    clientes = []
    for i in range(1, n + 1):
        tipo_rand = random.random()
        acum = 0
        tipo = "pessoa_fisica"
        for tipo_nome, prob in TIPOS_CLIENTE:
            acum += prob
            if tipo_rand <= acum:
                tipo = tipo_nome
                break

        clientes.append({
            "id":       f"CLI{i:04d}",
            "nome":     f"Cliente {i:04d}",
            "tipo":     tipo,
            "bairro":   random.choice(BAIRROS),
            "ativo":    "sim",
        })
    return clientes

# ── Sazonalidade ─────────────────────────────────────────────
def fator_sazonalidade(data: datetime) -> float:
    """
    Fatores que afetam vendas de materiais de construção:
    - Janeiro/Fevereiro: baixo (férias, carnaval)
    - Março/Abril: médio (início de obras pós-chuvas)
    - Maio/Junho: alto (seco, muitas obras)
    - Julho: médio (férias escolares)
    - Agosto/Setembro: alto (pico de obras)
    - Outubro/Novembro: médio (chuvas começando)
    - Dezembro: baixo (festas, fechamentos)
    """
    fatores_mes = {
        1: 0.65, 2: 0.60, 3: 0.85, 4: 0.90,
        5: 1.15, 6: 1.20, 7: 0.95, 8: 1.25,
        9: 1.20, 10: 1.00, 11: 0.90, 12: 0.70
    }

    fator = fatores_mes.get(data.month, 1.0)

    # Dia da semana — segunda e terça são mais fracos
    if data.weekday() == 0:  # segunda
        fator *= 0.80
    elif data.weekday() == 6:  # domingo
        fator *= 0.20
    elif data.weekday() == 5:  # sábado
        fator *= 0.70

    return fator

def probabilidade_compra(cliente: dict, data: datetime) -> float:
    """Probabilidade de um cliente comprar em determinado dia"""
    base = {
        "construtora":   0.15,  # compra quase todo dia
        "autonomo":      0.05,  # compra algumas vezes por semana
        "pessoa_fisica": 0.01,  # compra raramente
    }
    prob = base.get(cliente["tipo"], 0.01)
    return prob * fator_sazonalidade(data)

def gerar_itens_venda(venda_id: str, cliente: dict):
    """Gera itens de uma venda baseado no perfil do cliente"""
    n_itens = {
        "construtora":   random.randint(3, 8),
        "autonomo":      random.randint(2, 5),
        "pessoa_fisica": random.randint(1, 3),
    }.get(cliente["tipo"], 2)

    # Produtos com maior probabilidade de serem comprados juntos
    # baseado em conhecimento do domínio
    AFINIDADES = {
        "P001": ["P003", "P005", "P029"],  # cimento → areia, brita, argamassa
        "P009": ["P018", "P019"],           # ferro → disco de corte, rebolo
        "P016": ["P029", "P030"],           # rejunte → argamassa, gesso
        "P022": ["P023", "P024"],           # tubo PVC → tubo menor, joelho
        "P014": ["P015", "P017"],           # tinta latex → tinta acrílica, massa
    }

    produtos_escolhidos = []
    produto_inicial = random.choice(PRODUTOS)
    produtos_escolhidos.append(produto_inicial)

    # Adiciona produtos afins com probabilidade
    codigo_inicial = produto_inicial[0]
    if codigo_inicial in AFINIDADES and random.random() < 0.65:
        afins = AFINIDADES[codigo_inicial]
        produto_afim_code = random.choice(afins)
        produto_afim = next((p for p in PRODUTOS if p[0] == produto_afim_code), None)
        if produto_afim:
            produtos_escolhidos.append(produto_afim)

    # Completa com produtos aleatórios
    while len(produtos_escolhidos) < n_itens:
        p = random.choice(PRODUTOS)
        if p not in produtos_escolhidos:
            produtos_escolhidos.append(p)

    itens = []
    for prod in produtos_escolhidos:
        codigo, nome, cat, subcat, un, custo, preco = prod

        # Quantidade baseada no tipo de cliente
        if cliente["tipo"] == "construtora":
            qtd = random.randint(10, 100)
        elif cliente["tipo"] == "autonomo":
            qtd = random.randint(2, 20)
        else:
            qtd = random.randint(1, 5)

        # Desconto ocasional
        desconto_pct = 0
        if cliente["tipo"] == "construtora" and random.random() < 0.4:
            desconto_pct = random.choice([0.03, 0.05, 0.08, 0.10])
        elif cliente["tipo"] == "autonomo" and random.random() < 0.15:
            desconto_pct = random.choice([0.02, 0.03, 0.05])

        preco_unit = round(preco * (1 + random.uniform(-0.02, 0.02)), 2)
        desconto   = round(preco_unit * qtd * desconto_pct, 2)
        total      = round(preco_unit * qtd - desconto, 2)

        itens.append({
            "venda_id":       venda_id,
            "produto_key":    codigo,
            "descricao":      nome,
            "quantidade":     qtd,
            "unidade":        un,
            "preco_unitario": preco_unit,
            "desconto":       desconto,
            "total":          total,
        })

    return itens

# ── Geração principal ────────────────────────────────────────

def gerar_dados():
    print("Lume - Gerando dados sinteticos...")
    clientes    = gerar_clientes(150)
    vendas      = []
    itens_venda = []
    venda_num   = 1

    data_atual = START_DATE
    while data_atual <= END_DATE:
        for cliente in clientes:
            prob = probabilidade_compra(cliente, data_atual)
            if random.random() < prob:
                venda_id = f"V{venda_num:06d}"

                itens = gerar_itens_venda(venda_id, cliente)
                total_venda    = sum(i["total"] for i in itens)
                total_desconto = sum(i["desconto"] for i in itens)

                hora = random.randint(7, 18)
                minuto = random.randint(0, 59)
                data_hora = data_atual.replace(hour=hora, minute=minuto)

                vendas.append({
                    "id":       venda_id,
                    "data":     data_hora.strftime("%Y-%m-%d %H:%M:%S"),
                    "cliente":  cliente["id"],
                    "vendedor": f"VEN{random.randint(1, 4):02d}",
                    "total":    round(total_venda, 2),
                    "desconto": round(total_desconto, 2),
                    "status":   "concluida",
                    "canal":    random.choice(["balcao", "balcao", "balcao", "telefone"]),
                })

                itens_venda.extend(itens)
                venda_num += 1

        data_atual += timedelta(days=1)

    # ── Salva os CSVs ────────────────────────────────────────
    salvar_csv("clientes_sinteticos.csv", clientes,
               ["id", "nome", "tipo", "bairro", "ativo"])

    salvar_csv("produtos_sinteticos.csv",
               [{"id": p[0], "nome": p[1], "categoria": p[2],
                 "subcategoria": p[3], "unidade": p[4],
                 "custo": p[5], "preco": p[6], "ativo": "sim"}
                for p in PRODUTOS],
               ["id", "nome", "categoria", "subcategoria",
                "unidade", "custo", "preco", "ativo"])

    salvar_csv("vendas_sinteticas.csv", vendas,
               ["id", "data", "cliente", "vendedor",
                "total", "desconto", "status", "canal"])

    salvar_csv("itens_venda_sinteticos.csv", itens_venda,
               ["venda_id", "produto_key", "descricao", "quantidade",
                "unidade", "preco_unitario", "desconto", "total"])

    print(f"OK! Geracao concluida:")
    print(f"   {len(clientes)} clientes")
    print(f"   {len(PRODUTOS)} produtos")
    print(f"   {len(vendas)} vendas")
    print(f"   {len(itens_venda)} itens de venda")

def salvar_csv(nome, dados, campos):
    path = os.path.join(OUTPUT_DIR, nome)
    with open(path, "w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=campos)
        writer.writeheader()
        writer.writerows(dados)
    print(f"   >> {nome} salvo ({len(dados)} registros)")

if __name__ == "__main__":
    gerar_dados()