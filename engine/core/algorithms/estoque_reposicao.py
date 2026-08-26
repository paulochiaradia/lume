import pandas as pd
import math
import logging
from sqlalchemy import text

log = logging.getLogger(__name__)

LEAD_TIME_PADRAO = 7
DIAS_COBERTURA_ALVO = 30
FATOR_SEGURANCA = 1.2

def calcular_reposicao(df_produtos: pd.DataFrame, df_estoque: pd.DataFrame, df_forecast: pd.DataFrame, df_abc: pd.DataFrame) -> pd.DataFrame:
    """
    Calcula a fila de reposição cruzando o estoque físico, forecast do Prophet e a Curva ABC.
    """
    if df_produtos.empty or df_estoque.empty:
        log.warning("Reposição: DataFrame de produtos ou estoque vazio")
        return pd.DataFrame()

    # Evita erros se as chaves estiverem no índice do DuckDB
    if "produto_key" not in df_produtos.columns: df_produtos = df_produtos.reset_index()
    if "produto_key" not in df_estoque.columns: df_estoque = df_estoque.reset_index()

    # 1. Mescla Produtos e Estoque pela chave do ERP
    df_base = pd.merge(
        df_produtos[["produto_key", "nome", "categoria"]],
        df_estoque[["produto_key", "quantidade"]],
        on="produto_key", how="inner"
    )

    # 2. Mescla com ABC para pegar a classe do produto
    if not df_abc.empty:
        if "produto_key" not in df_abc.columns: df_abc = df_abc.reset_index()
        # O abc_xyz.py geralmente gera a coluna 'classe'
        col_classe = "classe" if "classe" in df_abc.columns else "classe_abc" if "classe_abc" in df_abc.columns else None
        
        if col_classe:
            df_base = pd.merge(df_base, df_abc[["produto_key", col_classe]], on="produto_key", how="left")
            df_base["classe_abc"] = df_base[col_classe].fillna('C')
        else:
            df_base["classe_abc"] = 'C'
    else:
        df_base["classe_abc"] = 'C'

    # Renomeia para o padrão da nossa tabela final
    df_base.rename(columns={"quantidade": "estoque_atual"}, inplace=True)

    # 3. Processa o Forecast (Se existir)
    if not df_forecast.empty and "ds" in df_forecast.columns:
        if "produto_key" not in df_forecast.columns: df_forecast = df_forecast.reset_index()
        fk_col = "produto_key" if "produto_key" in df_forecast.columns else "produto_id" if "produto_id" in df_forecast.columns else None
        
        if fk_col:
            df_forecast["ds"] = pd.to_datetime(df_forecast["ds"], utc=True).dt.tz_localize(None)
            hoje = pd.Timestamp.today().normalize()
            df_futuro = df_forecast[df_forecast["ds"] >= hoje].copy()
            df_futuro["yhat"] = df_futuro["yhat"].clip(lower=0) 
            
            demanda = df_futuro.groupby(fk_col)["yhat"].mean().reset_index()
            demanda.rename(columns={"yhat": "demanda_diaria_media", fk_col: "produto_key"}, inplace=True)
        else:
            demanda = pd.DataFrame(columns=["produto_key", "demanda_diaria_media"])
    else:
        demanda = pd.DataFrame(columns=["produto_key", "demanda_diaria_media"])

    # 4. Junta tudo
    df = pd.merge(df_base, demanda, on="produto_key", how="left")
    df["demanda_diaria_media"] = df["demanda_diaria_media"].fillna(0)

    resultados = []
    
    # 5. Aplica a Regra de Negócio de Reposição
    for _, row in df.iterrows():
        demanda_media = row["demanda_diaria_media"]
        estoque_atual = row["estoque_atual"]
        
        if demanda_media <= 0:
            continue
            
        dias_ate_ruptura = math.floor(estoque_atual / demanda_media) if estoque_atual > 0 else 0
        demanda_periodo = demanda_media * (LEAD_TIME_PADRAO + DIAS_COBERTURA_ALVO)
        estoque_seguranca = demanda_media * LEAD_TIME_PADRAO * (FATOR_SEGURANCA - 1)
        
        qtd_sugerida = max(0, round((demanda_periodo + estoque_seguranca) - estoque_atual, 2))
        classe_abc = row["classe_abc"]
        
        if dias_ate_ruptura <= LEAD_TIME_PADRAO and classe_abc == 'A':
            urgencia = 1
        elif dias_ate_ruptura <= LEAD_TIME_PADRAO:
            urgencia = 2
        elif dias_ate_ruptura <= (LEAD_TIME_PADRAO + 5) or (classe_abc == 'A' and dias_ate_ruptura <= 15):
            urgencia = 3
        else:
            urgencia = 4
            
        if qtd_sugerida > 0 or urgencia <= 3:
            resultados.append({
                "id": str(row["produto_key"]), # Gravamos o produto_key como 'id' na tabela cache para cruzar no Go
                "nome": row["nome"],
                "categoria": row["categoria"] if pd.notna(row["categoria"]) else "Sem Categoria",
                "classe_abc": str(classe_abc)[0],
                "estoque_atual": float(estoque_atual),
                "demanda_prevista": round(float(demanda_media * DIAS_COBERTURA_ALVO), 2),
                "dias_ate_ruptura": int(dias_ate_ruptura),
                "quantidade_sugerida": float(qtd_sugerida),
                "urgencia": int(urgencia)
            })
            
    return pd.DataFrame(resultados)

def salvar_reposicao_postgres(client_key: str, df_reposicao: pd.DataFrame):
    if df_reposicao is None or df_reposicao.empty: return
    try:
        from core.db.postgres import get_engine
        engine = get_engine()
        schema = f"client_{client_key}"
        with engine.begin() as conn:
            conn.execute(text(f"TRUNCATE TABLE {schema}.estoque_reposicao_cache"))
            df_reposicao.to_sql("estoque_reposicao_cache", conn, schema=schema, if_exists="append", index=False)
        log.info(f"[{client_key}] Fila de Reposição salva no PostgreSQL.")
    except Exception as e:
        log.error(f"[{client_key}] Erro ao salvar reposição: {e}")