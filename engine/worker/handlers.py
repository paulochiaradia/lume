import logging
from worker.job_runner import register

log = logging.getLogger(__name__)

@register("rfm")
def handle_rfm(client_key: str):
    """Executa análise RFM"""
    from core.algorithms.rfm import calcular_rfm
    from segments import get_engine
    engine = get_engine(client_key)
    df_vendas = engine.get_vendas()
    resultado = calcular_rfm(df_vendas)
    engine.save("rfm_resultado", resultado)
    log.info(f"RFM concluido para {client_key}: {len(resultado)} clientes")

@register("abc_xyz")
def handle_abc(client_key: str):
    """Executa análise ABC"""
    from core.algorithms.abc_xyz import calcular_abc
    from segments import get_engine
    engine = get_engine(client_key)
    df_itens = engine.get_itens_venda()
    resultado = calcular_abc(df_itens)
    engine.save("abc_resultado", resultado)
    log.info(f"ABC concluido para {client_key}: {len(resultado)} produtos")