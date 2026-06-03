import logging

log = logging.getLogger(__name__)

# Mapeamento de job_type para função
JOB_HANDLERS = {}

def register(job_type: str):
    """Decorator para registrar handlers de job"""
    def decorator(func):
        JOB_HANDLERS[job_type] = func
        return func
    return decorator

def run_job(client_key: str, job_type: str):
    """Executa o handler correto para o job_type"""
    handler = JOB_HANDLERS.get(job_type)
    if not handler:
        raise ValueError(f"job_type '{job_type}' não registrado")
    handler(client_key)

# Importa os handlers para registrá-los
from worker import handlers  # noqa