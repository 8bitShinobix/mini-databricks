class MiniDatabricksError(Exception):
    pass


class AuthenticationError(MiniDatabricksError):
    pass


class NotFoundError(MiniDatabricksError):
    pass


class APIError(MiniDatabricksError):
    def __init__(self, message: str, status_code: int):
        super().__init__(message)
        self.status_code = status_code
