import os
import pandas as pd
from sqlalchemy import create_engine, text
from contextlib import contextmanager

def get_dsn() -> str:
    dsn = os.getenv("POSTGRES_DSN")
    if not dsn:
        raise ValueError("POSTGRES_DSN não definida")
    return dsn

def get_engine():
    return create_engine(get_dsn())

def read_table(client_key: str, table: str) -> pd.DataFrame:
    """Lê uma tabela do schema do cliente"""
    engine = get_engine()
    schema = f"client_{client_key}"
    with engine.connect() as conn:
        return pd.read_sql(
            text(f"SELECT * FROM {schema}.{table}"),
            conn
        )

def read_query(client_key: str, query: str, params: dict = None) -> pd.DataFrame:
    """Executa uma query no contexto do cliente"""
    engine = get_engine()
    with engine.connect() as conn:
        return pd.read_sql(text(query), conn, params=params)

def get_active_clients() -> list:
    """Retorna lista de client_keys ativos"""
    engine = get_engine()
    with engine.connect() as conn:
        result = conn.execute(text(
            "SELECT client_key FROM lume_system.clients WHERE active = true"
        ))
        return [row[0] for row in result]

def get_pending_jobs() -> list:
    """Retorna jobs pendentes da fila"""
    engine = get_engine()
    with engine.connect() as conn:
        result = conn.execute(text("""
            SELECT id, client_id, job_type, payload
            FROM lume_system.jobs
            WHERE status = 'pending'
            ORDER BY created_at
            LIMIT 10
        """))
        return [dict(row._mapping) for row in result]

def update_job_status(job_id: str, status: str, error: str = None):
    """Atualiza o status de um job"""
    engine = get_engine()
    with engine.connect() as conn:
        conn.execute(text("""
            UPDATE lume_system.jobs
            SET status = :status,
                error_message = :error,
                updated_at = NOW()
            WHERE id = :id
        """), {"status": status, "error": error, "id": job_id})
        conn.commit()