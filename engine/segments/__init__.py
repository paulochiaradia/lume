from core.engine.base import SegmentEngine
from core.db.postgres import read_query

def get_engine(client_key: str) -> SegmentEngine:
    """
    Retorna o engine correto para o segmento do cliente.
    Busca o segmento na tabela de clientes e instancia o engine certo.
    """
    from sqlalchemy import text
    from core.db.postgres import get_engine as get_db_engine

    db_engine = get_db_engine()
    with db_engine.connect() as conn:
        result = conn.execute(text(
            "SELECT segment FROM lume_system.clients WHERE client_key = :key"
        ), {"key": client_key})
        row = result.fetchone()
        segment = row[0] if row else "construcao"

    if segment == "construcao":
        from segments.construcao.engine import ConstrucaoEngine
        return ConstrucaoEngine(client_key)

    # Fallback para o engine base
    return _BaseEngine(client_key)

class _BaseEngine(SegmentEngine):
    pass