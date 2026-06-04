import re
import logging

log = logging.getLogger(__name__)

def sanitize(data: dict) -> dict:
    """
    Remove dados pessoais identificáveis antes de enviar para APIs externas.
    Substitui por placeholders genéricos.
    LGPD — nunca enviamos CPF, nome real, telefone ou endereço para LLMs.
    """
    sanitized = {}

    for key, value in data.items():
        if isinstance(value, str):
            # Remove CPF/CNPJ
            value = re.sub(r'\d{3}\.\d{3}\.\d{3}-\d{2}', '[CPF]', value)
            value = re.sub(r'\d{2}\.\d{3}\.\d{3}/\d{4}-\d{2}', '[CNPJ]', value)

            # Remove telefones
            value = re.sub(r'\(?\d{2}\)?\s?\d{4,5}-?\d{4}', '[TELEFONE]', value)

            # Remove e-mails
            value = re.sub(r'[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}', '[EMAIL]', value)

            sanitized[key] = value
        else:
            sanitized[key] = value

    return sanitized