import logging
import time
from core.db.postgres import get_pending_jobs, update_job_status, get_engine
from sqlalchemy import text

log = logging.getLogger(__name__)

def get_client_key(client_id: str) -> str:
    """Busca o client_key pelo client_id"""
    engine = get_engine()
    with engine.connect() as conn:
        result = conn.execute(text(
            "SELECT client_key FROM lume_system.clients WHERE id = :id"
        ), {"id": client_id})
        row = result.fetchone()
        return row[0] if row else None

def process_job(job: dict):
    """Processa um job da fila"""
    job_id    = str(job["id"])
    client_id = str(job["client_id"])
    job_type  = job["job_type"]

    client_key = get_client_key(client_id)
    if not client_key:
        update_job_status(job_id, "error", "cliente não encontrado")
        return

    log.info(f"processando job {job_type} para cliente {client_key}")
    update_job_status(job_id, "running")

    try:
        from worker.job_runner import run_job
        run_job(client_key, job_type)
        update_job_status(job_id, "done")
        log.info(f"job {job_type} concluido para {client_key}")

    except Exception as e:
        log.error(f"erro no job {job_type} para {client_key}: {e}")
        update_job_status(job_id, "error", str(e))

def start():
    """Loop principal do consumer"""
    log.info("worker: aguardando jobs...")

    while True:
        try:
            jobs = get_pending_jobs()
            for job in jobs:
                process_job(job)
        except Exception as e:
            log.error(f"erro no consumer: {e}")

        time.sleep(5)