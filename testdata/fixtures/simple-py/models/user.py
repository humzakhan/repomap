"""User model definition."""


class User:
    """Represents a user in the system."""

    def __init__(self, id, name, email):
        self.id = id
        self.name = name
        self.email = email

    def to_dict(self):
        """Convert user to dictionary."""
        return {"id": self.id, "name": self.name, "email": self.email}
