import logging
from abc import ABC, abstractmethod
from core.db.postgres import read_query
from core.db.duckdb import write_result, read_result

log = logging.getLogger(__name__)

class SegmentEngine(ABC):
    """
    Classe base para engines de segmento.
    Cada segmento herda esta classe e sobrescreve
    apenas o que é específico do seu domínio.
    """

    def __init__(self, client_key: str):
        self.client_key = client_key
        self.schema = f"client_{client_key}"

    # ── Métodos que os segmentos podem sobrescrever ──────────

    def get_churn_window_days(self) -> int:
        """Janela de dias sem compra para considerar churn"""
        return 45

    def get_abc_threshold_a(self) -> float:
        """Percentual acumulado para classe A"""
        return 0.80

    def get_abc_threshold_b(self) -> float:
        """Percentual acumulado para classe B"""
        return 0.95

    # ── Métodos genéricos — iguais para todos os segmentos ───

    def read(self, query: str) -> object:
        """Lê dados do PostgreSQL"""
        return read_query(self.client_key, query)

    def save(self, table: str, df: object):
        """Salva resultado no DuckDB"""
        write_result(self.client_key, table, df)

    def load(self, table: str) -> object:
        """Carrega resultado do DuckDB"""
        return read_result(self.client_key, table)

    def get_vendas(self) -> object:
        """Retorna todas as vendas do cliente"""
        return self.read(f"""
            SELECT v.id, v.venda_key, v.data_venda,
                   v.cliente_id, v.vendedor_id,
                   v.total, v.desconto, v.status
            FROM {self.schema}.vendas v
            WHERE v.status = 'concluida'
            ORDER BY v.data_venda
        """)

    def get_itens_venda(self) -> object:
        """Retorna todos os itens de venda"""
        return self.read(f"""
            SELECT iv.venda_id, iv.produto_key,
                   iv.quantidade, iv.total,
                   v.data_venda, v.cliente_id
            FROM {self.schema}.itens_venda iv
            JOIN {self.schema}.vendas v ON v.id = iv.venda_id
            WHERE v.status = 'concluida'
        """)

    def get_clientes(self) -> object:
        """Retorna todos os clientes ativos"""
        return self.read(f"""
            SELECT cliente_key, nome, tipo, cidade, bairro
            FROM {self.schema}.clientes
            WHERE ativo = true
        """)

    def get_produtos(self) -> object:
        """Retorna todos os produtos ativos"""
        return self.read(f"""
            SELECT produto_key, nome, categoria,
                   subcategoria, preco_custo, preco_venda
            FROM {self.schema}.produtos
            WHERE ativo = true
        """)