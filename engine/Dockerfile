FROM python:3.12-slim

WORKDIR /app

# Dependências do sistema necessárias para o Prophet/cmdstan
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

# Instala o cmdstan após o prophet
RUN python -c "import cmdstanpy; cmdstanpy.install_cmdstan()"

COPY . .

CMD ["python", "-m", "worker.main"]