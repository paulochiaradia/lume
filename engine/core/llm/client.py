import os
import logging
import json

log = logging.getLogger(__name__)


class LLMClient:
    """
    Cliente abstrato para modelos de linguagem.
    Em modo mock retorna respostas simuladas sem custo.
    Em produção chama a API real configurada.
    """

    def __init__(self):
        self.provider = os.getenv("LLM_PROVIDER", "mock")
        self.api_key  = os.getenv("OPENAI_API_KEY") or os.getenv("GEMINI_API_KEY")

        if self.provider != "mock" and not self.api_key:
            log.warning("LLM_PROVIDER definido mas sem API key — usando mock")
            self.provider = "mock"

        log.info(f"LLM client iniciado — provider: {self.provider}")

    def complete(self, prompt: str, max_tokens: int = 500) -> str:
        """
        Envia um prompt e retorna a resposta em texto.
        """
        if self.provider == "mock":
            return self._mock_response(prompt)

        if self.provider == "openai":
            return self._openai(prompt, max_tokens)

        if self.provider == "gemini":
            return self._gemini(prompt, max_tokens)

        return self._mock_response(prompt)

    def _mock_response(self, prompt: str) -> str:
        """
        Resposta simulada baseada em palavras-chave do prompt.
        Permite desenvolver e testar sem custo de API.
        """
        prompt_lower = prompt.lower()

        if "insight" in prompt_lower and "venda" in prompt_lower:
            return (
                "As vendas apresentam tendência de crescimento nos últimos 30 dias. "
                "O ticket médio aumentou 8% em relação ao período anterior. "
                "Recomenda-se manter o estoque dos produtos classe A reforçado "
                "para aproveitar o momento favorável."
            )

        if "churn" in prompt_lower or "risco" in prompt_lower:
            return (
                "Identificamos clientes com alto valor histórico que reduziram "
                "a frequência de compras. Sugerimos contato proativo esta semana "
                "com oferta personalizada baseada no histórico de produtos comprados."
            )

        if "estoque" in prompt_lower or "reposição" in prompt_lower:
            return (
                "O nível de estoque dos produtos classe A está abaixo do ponto "
                "de reposição recomendado. Considerando a sazonalidade do período, "
                "sugere-se antecipar o pedido ao fornecedor em pelo menos 7 dias."
            )

        if "abc" in prompt_lower or "produto" in prompt_lower:
            return (
                "Os produtos classe A representam 80% do faturamento com apenas "
                "20% do catálogo. Garanta disponibilidade permanente desses itens "
                "e avalie descontinuar produtos classe C com baixo giro."
            )

        if "relatorio" in prompt_lower or "relatório" in prompt_lower:
            return (
                "Semana com desempenho dentro do esperado para o período. "
                "Destaques: crescimento de 12% no faturamento vs semana anterior, "
                "redução de 3% nos descontos concedidos. "
                "Ponto de atenção: 5 clientes estratégicos sem compra há mais de 30 dias."
            )

        return (
            "Análise concluída. Os dados indicam oportunidades de melhoria "
            "na gestão de estoque e relacionamento com clientes estratégicos. "
            "Consulte os módulos específicos para detalhes e recomendações."
        )

    def _openai(self, prompt: str, max_tokens: int) -> str:
        """Chama a API da OpenAI"""
        try:
            import openai
            client = openai.OpenAI(api_key=self.api_key)
            response = client.chat.completions.create(
                model="gpt-4o-mini",
                messages=[
                    {
                        "role": "system",
                        "content": (
                            "Você é um analista de dados especializado em varejo "
                            "de materiais de construção. Forneça insights práticos "
                            "e acionáveis em português brasileiro. Seja direto e objetivo."
                        )
                    },
                    {"role": "user", "content": prompt}
                ],
                max_tokens=max_tokens,
                temperature=0.3,
            )
            return response.choices[0].message.content
        except Exception as e:
            log.error(f"erro na API OpenAI: {e}")
            return self._mock_response(prompt)

    def _gemini(self, prompt: str, max_tokens: int) -> str:
        """Chama a API do Gemini"""
        try:
            import google.generativeai as genai
            genai.configure(api_key=self.api_key)
            model = genai.GenerativeModel("gemini-1.5-flash")
            response = model.generate_content(prompt)
            return response.text
        except Exception as e:
            log.error(f"erro na API Gemini: {e}")
            return self._mock_response(prompt)


# Instância global — singleton
_client = None

def get_llm_client() -> LLMClient:
    global _client
    if _client is None:
        _client = LLMClient()
    return _client