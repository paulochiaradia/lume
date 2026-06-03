import logging
import time

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(message)s"
)

log = logging.getLogger(__name__)

def main():
    print("╔══════════════════════════════════════╗")
    print("║     Lume — Engine Worker             ║")
    print("╚══════════════════════════════════════╝")

    log.info("engine worker iniciado")

    from worker.job_consumer import start
    start()

if __name__ == "__main__":
    main()