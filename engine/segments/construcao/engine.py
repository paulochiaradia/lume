from core.engine.base import SegmentEngine

class ConstrucaoEngine(SegmentEngine):
    """
    Engine específico para lojas de materiais de construção.
    Sobrescreve apenas o que é diferente do engine base.
    """

    def get_churn_window_days(self) -> int:
        # Clientes de construção que ficam 45 dias sem comprar
        # provavelmente estão comprando do concorrente
        return 45

    def get_abc_threshold_a(self) -> float:
        return 0.80

    def get_abc_threshold_b(self) -> float:
        return 0.95