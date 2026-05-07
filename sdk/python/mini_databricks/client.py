from typing import Optional

import requests

from .exceptions import APIError, AuthenticationError, NotFoundError


class Client:
    def __init__(self, base_url: str, token: Optional[str] = None):
        self.base_url = base_url.rstrip("/")
        self.token = token
        self.session = requests.Session()

    def login(self, email: str, password: str) -> str:
        response = self.session.post(
            f"{self.base_url}/auth/login", json={"email": email, "password": password}
        )
        data = self._handle_response(response)
        self.token = data["token"]
        return self.token

    def _headers(self) -> dict:
        if not self.token:
            raise AuthenticationError("not authenticated — call login() first")
        return {"Authorization": f"Bearer {self.token}"}

    def _handle_response(self, response: requests.Response) -> dict:
        if response.status_code == 401:
            raise AuthenticationError("invalid credentials or token expired")
        if response.status_code == 404:
            raise NotFoundError("resource not found")
        if not response.ok:
            raise APIError(
                response.json().get("error", "unknown error"), response.status_code
            )
        return response.json()

    def get(self, path: str, params: Optional[dict] = None) -> dict:
        response = self.session.get(
            f"{self.base_url}{path}", headers=self._headers(), params=params
        )
        return self._handle_response(response)

    def post(self, path: str, body: Optional[dict] = None) -> dict:
        response = self.session.post(
            f"{self.base_url}{path}", headers=self._headers(), json=body
        )
        return self._handle_response(response)

    def delete(self, path: str) -> dict:
        response = self.session.delete(
            f"{self.base_url}{path}", headers=self._headers()
        )
        return self._handle_response(response)
