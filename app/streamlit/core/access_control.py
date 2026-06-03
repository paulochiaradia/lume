# Páginas permitidas por role
PAGES_BY_ROLE = {
    "admin": [
        "🏠 Home",
        "📊 Vendas",
        "📦 Estoque",
        "👥 Clientes",
        "🛒 Produtos",
        "💰 Preço & Margem",
        "🔍 Anomalias",
        "📄 Relatórios",
    ],
    "gerente": [
        "🏠 Home",
        "📊 Vendas",
        "🔍 Anomalias",
        "📄 Relatórios",
    ],
    "compras": [
        "🏠 Home",
        "📦 Estoque",
        "📄 Relatórios",
    ],
    "viewer": [
        "🏠 Home",
        "📊 Vendas",
    ],
}

# Mapeamento de página para módulo
PAGE_TO_MODULE = {
    "🏠 Home":          "home",
    "📊 Vendas":        "vendas",
    "📦 Estoque":       "estoque",
    "👥 Clientes":      "clientes",
    "🛒 Produtos":      "produtos",
    "💰 Preço & Margem": "preco",
    "🔍 Anomalias":     "anomalias",
    "📄 Relatórios":    "relatorios",
}

def get_allowed_pages(role: str) -> list:
    """Retorna as páginas permitidas para o role"""
    return PAGES_BY_ROLE.get(role, PAGES_BY_ROLE["viewer"])

def get_module(page: str) -> str:
    """Retorna o módulo correspondente à página"""
    return PAGE_TO_MODULE.get(page, "home")

def can_access(role: str, page: str) -> bool:
    """Verifica se o role tem acesso à página"""
    return page in get_allowed_pages(role)