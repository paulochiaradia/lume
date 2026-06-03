import os
import duckdb
import pandas as pd

def get_duckdb_path(client_key: str) -> str:
    """Retorna o caminho do arquivo DuckDB do cliente"""
    base_path = os.getenv("DUCKDB_PATH", "/app/data/duckdb")
    os.makedirs(base_path, exist_ok=True)
    return os.path.join(base_path, f"{client_key}.duckdb")

def get_connection(client_key: str) -> duckdb.DuckDBPyConnection:
    """Abre conexão com o DuckDB do cliente"""
    path = get_duckdb_path(client_key)
    return duckdb.connect(path)

def write_result(client_key: str, table: str, df: pd.DataFrame):
    """Grava um DataFrame no DuckDB do cliente"""
    if df is None or df.empty:
        return

    conn = get_connection(client_key)
    try:
        conn.execute(f"DROP TABLE IF EXISTS {table}")
        conn.execute(f"CREATE TABLE {table} AS SELECT * FROM df")
        conn.commit()
    finally:
        conn.close()

def read_result(client_key: str, table: str) -> pd.DataFrame:
    """Lê uma tabela de resultados do DuckDB"""
    conn = get_connection(client_key)
    try:
        result = conn.execute(f"SELECT * FROM {table}").df()
        return result
    except Exception:
        return pd.DataFrame()
    finally:
        conn.close()

def table_exists(client_key: str, table: str) -> bool:
    """Verifica se uma tabela existe no DuckDB"""
    conn = get_connection(client_key)
    try:
        conn.execute(f"SELECT 1 FROM {table} LIMIT 1")
        return True
    except Exception:
        return False
    finally:
        conn.close()