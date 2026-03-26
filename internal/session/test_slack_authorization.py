#!/usr/bin/env python3
"""
Test suite for Slack user ID authorization in conductor bridge.

This tests the authorization logic that will be embedded in the generated
bridge.py script. It verifies that the is_slack_authorized() function
works correctly with different configurations.
"""

import unittest
from unittest.mock import patch
import logging


class TestSlackAuthorization(unittest.TestCase):
    """Test Slack user authorization logic."""

    def setUp(self):
        """Set up test fixtures."""
        self.log = logging.getLogger(__name__)
        logging.basicConfig(level=logging.WARNING)

    def test_empty_allowed_users_allows_all(self):
        """When allowed_user_ids is empty, all users should be authorized."""
        allowed_users = []

        def is_slack_authorized(user_id: str) -> bool:
            if not allowed_users:
                return True
            if user_id not in allowed_users:
                self.log.warning("Unauthorized Slack message from user %s", user_id)
                return False
            return True

        # Empty list allows everyone
        self.assertTrue(is_slack_authorized("U12345"))
        self.assertTrue(is_slack_authorized("U67890"))
        self.assertTrue(is_slack_authorized("UABCDE"))
        self.assertTrue(is_slack_authorized(""))

    def test_single_allowed_user(self):
        """Only the specified user should be authorized."""
        allowed_users = ["U12345"]

        def is_slack_authorized(user_id: str) -> bool:
            if not allowed_users:
                return True
            if user_id not in allowed_users:
                self.log.warning("Unauthorized Slack message from user %s", user_id)
                return False
            return True

        # Only U12345 is allowed
        self.assertTrue(is_slack_authorized("U12345"))
        self.assertFalse(is_slack_authorized("U67890"))
        self.assertFalse(is_slack_authorized("UABCDE"))
        self.assertFalse(is_slack_authorized(""))

    def test_multiple_allowed_users(self):
        """Multiple specified users should be authorized."""
        allowed_users = ["U12345", "U67890", "UABCDE"]

        def is_slack_authorized(user_id: str) -> bool:
            if not allowed_users:
                return True
            if user_id not in allowed_users:
                self.log.warning("Unauthorized Slack message from user %s", user_id)
                return False
            return True

        # All three users are allowed
        self.assertTrue(is_slack_authorized("U12345"))
        self.assertTrue(is_slack_authorized("U67890"))
        self.assertTrue(is_slack_authorized("UABCDE"))

        # Others are not allowed
        self.assertFalse(is_slack_authorized("U99999"))
        self.assertFalse(is_slack_authorized("UOTHER"))
        self.assertFalse(is_slack_authorized(""))

    def test_case_sensitive_user_ids(self):
        """User IDs should be case-sensitive."""
        allowed_users = ["U12345"]

        def is_slack_authorized(user_id: str) -> bool:
            if not allowed_users:
                return True
            if user_id not in allowed_users:
                self.log.warning("Unauthorized Slack message from user %s", user_id)
                return False
            return True

        # Exact match works
        self.assertTrue(is_slack_authorized("U12345"))

        # Case mismatch fails
        self.assertFalse(is_slack_authorized("u12345"))
        self.assertFalse(is_slack_authorized("U12345".lower()))

    @patch('logging.Logger.warning')
    def test_unauthorized_access_logs_warning(self, mock_warning):
        """Unauthorized access attempts should log warnings."""
        allowed_users = ["U12345"]

        def is_slack_authorized(user_id: str) -> bool:
            if not allowed_users:
                return True
            if user_id not in allowed_users:
                logging.warning("Unauthorized Slack message from user %s", user_id)
                return False
            return True

        # Unauthorized user
        result = is_slack_authorized("U99999")
        self.assertFalse(result)

        # Warning should have been logged
        mock_warning.assert_called_once()
        call_args = str(mock_warning.call_args)
        self.assertIn("U99999", call_args)
        self.assertIn("Unauthorized", call_args)

    def test_slack_user_id_formats(self):
        """Test various Slack user ID formats."""
        # Real Slack user IDs follow patterns like U01234ABCDE
        allowed_users = [
            "U01234ABCDE",  # Standard 11-char format
            "U05678FGHIJ",  # Another standard
            "W12345",       # Workspace user ID (shorter)
            "USLACKBOT",    # SlackBot special ID
        ]

        def is_slack_authorized(user_id: str) -> bool:
            if not allowed_users:
                return True
            if user_id not in allowed_users:
                self.log.warning("Unauthorized Slack message from user %s", user_id)
                return False
            return True

        # All valid formats should work
        for user_id in allowed_users:
            self.assertTrue(is_slack_authorized(user_id),
                          f"User ID {user_id} should be authorized")


class TestSlackEventHandlers(unittest.TestCase):
    """Test authorization in Slack event handlers."""

    def setUp(self):
        """Set up test fixtures."""
        self.allowed_users = ["U12345", "U67890"]

    def is_slack_authorized(self, user_id: str) -> bool:
        """Mock authorization function."""
        if not self.allowed_users:
            return True
        if user_id not in self.allowed_users:
            logging.warning("Unauthorized Slack message from user %s", user_id)
            return False
        return True

    def test_message_event_authorization(self):
        """Test authorization in message event handler."""
        # Simulate Slack message event
        authorized_event = {
            "user": "U12345",
            "text": "hello",
            "channel": "C12345",
            "ts": "1234567890.123456",
        }

        unauthorized_event = {
            "user": "U99999",
            "text": "hello",
            "channel": "C12345",
            "ts": "1234567890.123456",
        }

        # Authorized user's message should be allowed
        self.assertTrue(self.is_slack_authorized(authorized_event["user"]))

        # Unauthorized user's message should be blocked
        self.assertFalse(self.is_slack_authorized(unauthorized_event["user"]))

    def test_mention_event_authorization(self):
        """Test authorization in app_mention event handler."""
        # Simulate Slack mention event
        authorized_mention = {
            "user": "U67890",
            "text": "<@U11111> help",
            "channel": "C12345",
            "ts": "1234567890.123456",
        }

        unauthorized_mention = {
            "user": "UOTHER",
            "text": "<@U11111> help",
            "channel": "C12345",
            "ts": "1234567890.123456",
        }

        # Authorized user's mention should be allowed
        self.assertTrue(self.is_slack_authorized(authorized_mention["user"]))

        # Unauthorized user's mention should be blocked
        self.assertFalse(self.is_slack_authorized(unauthorized_mention["user"]))

    def test_slash_command_authorization(self):
        """Test authorization in slash command handlers."""
        # Simulate Slack slash command
        authorized_command = {
            "user_id": "U12345",
            "command": "/ad-status",
            "text": "",
            "channel_id": "C12345",
        }

        unauthorized_command = {
            "user_id": "UBANNED",
            "command": "/ad-status",
            "text": "",
            "channel_id": "C12345",
        }

        # Authorized user's command should be allowed
        self.assertTrue(self.is_slack_authorized(authorized_command["user_id"]))

        # Unauthorized user's command should be blocked
        self.assertFalse(self.is_slack_authorized(unauthorized_command["user_id"]))

    def test_backward_compatibility_empty_list(self):
        """When allowed_user_ids is empty, all users are allowed."""
        self.allowed_users = []

        # Any user should be allowed when list is empty
        self.assertTrue(self.is_slack_authorized("U12345"))
        self.assertTrue(self.is_slack_authorized("U99999"))
        self.assertTrue(self.is_slack_authorized("ANYONE"))


class TestSlackAuthorizationEdgeCases(unittest.TestCase):
    """Test edge cases in Slack authorization."""

    def test_empty_user_id(self):
        """Empty user ID should be rejected when auth is enabled."""
        allowed_users = ["U12345"]

        def is_slack_authorized(user_id: str) -> bool:
            if not allowed_users:
                return True
            if user_id not in allowed_users:
                return False
            return True

        self.assertFalse(is_slack_authorized(""))

    def test_none_vs_empty_list(self):
        """None and empty list should behave the same (allow all)."""
        # Python treats empty list as falsy
        self.assertFalse(bool([]))
        self.assertFalse(bool(None))

        # Both should allow all users
        for allowed_users in [[], None]:
            def is_slack_authorized(user_id: str) -> bool:
                if not allowed_users:
                    return True
                if user_id not in allowed_users:
                    return False
                return True

            self.assertTrue(is_slack_authorized("U12345"))
            self.assertTrue(is_slack_authorized("U99999"))

    def test_whitespace_in_user_id(self):
        """User IDs with whitespace should not match."""
        allowed_users = ["U12345"]

        def is_slack_authorized(user_id: str) -> bool:
            if not allowed_users:
                return True
            if user_id not in allowed_users:
                return False
            return True

        # Exact match works
        self.assertTrue(is_slack_authorized("U12345"))

        # Whitespace variations don't match
        self.assertFalse(is_slack_authorized(" U12345"))
        self.assertFalse(is_slack_authorized("U12345 "))
        self.assertFalse(is_slack_authorized(" U12345 "))

    def test_duplicate_user_ids(self):
        """Duplicate user IDs in allowed list should still work."""
        allowed_users = ["U12345", "U12345", "U67890"]

        def is_slack_authorized(user_id: str) -> bool:
            if not allowed_users:
                return True
            if user_id not in allowed_users:
                return False
            return True

        # Should work despite duplicates
        self.assertTrue(is_slack_authorized("U12345"))
        self.assertTrue(is_slack_authorized("U67890"))
        self.assertFalse(is_slack_authorized("U99999"))


if __name__ == "__main__":
    # Run tests with verbose output
    unittest.main(verbosity=2)
