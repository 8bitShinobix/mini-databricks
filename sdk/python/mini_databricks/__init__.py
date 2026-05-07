from typing import Optional

from .client import Client
from .datasets import DatasetsAPI
from .jobs import JobsAPI


class MiniDatabricks:
    def __init__(self, base_url: str, token: Optional[str] = None):
        self.client = Client(base_url, token)
        self.jobs = JobsAPI(self.client)
        self.datasets = DatasetsAPI(self.client)

    def login(self, email: str, password: str) -> str:
        return self.client.login(email, password)
