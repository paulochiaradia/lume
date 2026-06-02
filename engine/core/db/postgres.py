import os
import pandas as pd
from sqlalchemy import create_engine, text

def get_engine():
    dsn = os.getenv("POSTGRES_DSN")
    if not dsn:
        raise ValueError("POSTGRES_DSN não definida")
    return create_engine(dsn)

def read_table(client_key: str, table: str) -> pd.DataFrame:
    """Lê uma tabela do schema do cliente e retorna um DataFrame"""
    engine = get_engine()
    schema = f"client_{client_key}"
    query = f"SELECT * FROM {schema}.{table}"
    with engine.connect() as conn:
        return pd.read_sql(text(query), conn)

def read_query(client_key: str, query: str) -> pd.DataFrame:
    """Executa uma query no schema do cliente e retorna um DataFrame"""
    engine = get_engine()
    with engine.connect() as conn:
        return pd.read_sql(text(query), conn)