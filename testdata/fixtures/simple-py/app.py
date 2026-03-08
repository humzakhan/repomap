"""Main application module."""

from .services.auth import AuthService
from .models.user import User

class Application:
    """The main application class."""

    def __init__(self, config):
        self.config = config
        self.auth = AuthService()

    def run(self):
        """Start the application."""
        print(f"Running on {self.config['host']}:{self.config['port']}")


def create_app(config=None):
    """Factory function to create an Application instance."""
    if config is None:
        config = {"host": "localhost", "port": 8000}
    return Application(config)


def _internal_helper():
    """This is a private helper."""
    pass
