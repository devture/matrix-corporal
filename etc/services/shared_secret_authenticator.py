# -*- coding: utf-8 -*-
#
# Shared Secret Authenticator module for Matrix Synapse
# Copyright (C) 2018 Slavi Pantaleev
#
# https://devture.com/
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as
# published by the Free Software Foundation, either version 3 of the
# License, or (at your option) any later version.

# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU Affero General Public License for more details.

# You should have received a copy of the GNU Affero General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.
#
from typing import Awaitable, Callable, Optional, Tuple

import hashlib
import hmac
import logging

import synapse
from synapse import module_api

logger = logging.getLogger(__name__)

class SharedSecretAuthProvider:
    def __init__(self, config: dict, api: module_api):
        for k in ('shared_secret',):
            if k not in config:
                raise KeyError('Required `{0}` configuration key not found'.format(k))

        m_login_password_support_enabled = bool(config['m_login_password_support_enabled']) if 'm_login_password_support_enabled' in config else False
        com_devture_shared_secret_auth_support_enabled = bool(config['com_devture_shared_secret_auth_support_enabled']) if 'com_devture_shared_secret_auth_support_enabled' in config else True

        self.api = api
        self.shared_secret = config['shared_secret']

        auth_checkers: Optional[Dict[Tuple[str, Tuple], CHECK_AUTH_CALLBACK]] = {}
        if com_devture_shared_secret_auth_support_enabled:
            auth_checkers[("com.devture.shared_secret_auth", ("token",))] = self.check_com_devture_shared_secret_auth
        if m_login_password_support_enabled:
            auth_checkers[("m.login.password", ("password",))] = self.check_m_login_password

        enabled_login_types = [k[0] for k in auth_checkers]

        if len(enabled_login_types) == 0:
            raise RuntimeError('At least one login type must be enabled')

        logger.info('Enabled login types: %s', enabled_login_types)

        api.register_password_auth_provider_callbacks(
            auth_checkers=auth_checkers,
        )

    async def check_com_devture_shared_secret_auth(
        self,
        username: str,
        login_type: str,
        login_dict: "synapse.module_api.JsonDict",
    ) -> Optional[
        Tuple[
            str,
            Optional[Callable[["synapse.module_api.LoginResponse"], Awaitable[None]]],
        ]
    ]:
        if login_type != "com.devture.shared_secret_auth":
            return None
        return await self._log_in_username_with_token("com.devture.shared_secret_auth", username, login_dict.get("token"))

    async def check_m_login_password(
        self,
        username: str,
        login_type: str,
        login_dict: "synapse.module_api.JsonDict",
    ) -> Optional[
        Tuple[
            str,
            Optional[Callable[["synapse.module_api.LoginResponse"], Awaitable[None]]],
        ]
    ]:
        if login_type != "m.login.password":
            return None
        return await self._log_in_username_with_token("m.login.password", username, login_dict.get("password"))

    async def _log_in_username_with_token(
        self,
        login_type: str,
        username: str,
        token: str,
    ) -> Optional[
        Tuple[
            str,
            Optional[Callable[["synapse.module_api.LoginResponse"], Awaitable[None]]],
        ]
    ]:
        logger.info('Authenticating user `%s` with login type `%s`', username, login_type)

        full_user_id = self.api.get_qualified_user_id(username)

        # The password (token) is supposed to be an HMAC of the full user id, keyed with the shared secret.
        given_hmac = token.encode('utf-8')

        h = hmac.new(self.shared_secret.encode('utf-8'), full_user_id.encode('utf-8'), hashlib.sha512)
        computed_hmac = h.hexdigest().encode('utf-8')

        if not hmac.compare_digest(computed_hmac, given_hmac):
            logger.info('Bad hmac value for user: %s', full_user_id)
            return None

        user_info = await self.api.get_userinfo_by_id(full_user_id)
        if user_info is None:
            logger.info('Refusing to authenticate missing user: %s', full_user_id)
            return None

        logger.info('Authenticated user: %s', full_user_id)

        return full_user_id, None
