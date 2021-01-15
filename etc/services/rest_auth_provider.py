# -*- coding: utf-8 -*-
#
# REST endpoint Authentication module for Matrix synapse
# Copyright (C) 2017 Kamax Sarl
#
# https://www.kamax.io/
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as
# published by the Free Software Foundation, either version 3 of the
# License, or (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program.  If not, see <http://www.gnu.org/licenses/>.
#

import logging
from twisted.internet import defer
import requests
import json
import time

logger = logging.getLogger(__name__)


class RestAuthProvider(object):

    def __init__(self, config, account_handler):
        self.account_handler = account_handler

        if not config.endpoint:
            raise RuntimeError('Missing endpoint config')

        self.endpoint = config.endpoint
        self.regLower = config.regLower
        self.config = config

        logger.info('Endpoint: %s', self.endpoint)
        logger.info('Enforce lowercase username during registration: %s', self.regLower)

    @defer.inlineCallbacks
    def check_password(self, user_id, password):
        logger.info("Got password check for " + user_id)
        data = {'user': {'id': user_id, 'password': password}}
        r = requests.post(self.endpoint + '/_matrix-internal/identity/v1/check_credentials', json=data)
        r.raise_for_status()
        r = r.json()
        if not r["auth"]:
            reason = "Invalid JSON data returned from REST endpoint"
            logger.warning(reason)
            raise RuntimeError(reason)

        auth = r["auth"]
        if not auth["success"]:
            logger.info("User not authenticated")
            defer.returnValue(False)

        localpart = user_id.split(":", 1)[0][1:]
        logger.info("User %s authenticated", user_id)

        registration = False
        if not (yield self.account_handler.check_user_exists(user_id)):
            logger.info("User %s does not exist yet, creating...", user_id)

            if localpart != localpart.lower() and self.regLower:
                logger.info('User %s was cannot be created due to username lowercase policy', localpart)
                defer.returnValue(False)

            user_id, access_token = (yield self.account_handler.register(localpart=localpart))
            registration = True
            logger.info("Registration based on REST data was successful for %s", user_id)
        else:
            logger.info("User %s already exists, registration skipped", user_id)

        if auth["profile"]:
            logger.info("Handling profile data")
            profile = auth["profile"]

            # fixme: temporary fix
            try:
                store = yield self.account_handler._hs.get_profile_handler().store  # for synapse >= 1.9.0
            except AttributeError:
                store = yield self.account_handler.hs.get_profile_handler().store   # for synapse < 1.9.0

            if "display_name" in profile and ((registration and self.config.setNameOnRegister) or (self.config.setNameOnLogin)):
                display_name = profile["display_name"]
                logger.info("Setting display name to '%s' based on profile data", display_name)
                yield store.set_profile_displayname(localpart, display_name)
            else:
                logger.info("Display name was not set because it was not given or policy restricted it")

            if (self.config.updateThreepid):
                if "three_pids" in profile:
                    logger.info("Handling 3PIDs")

                    external_3pids = []
                    for threepid in profile["three_pids"]:
                        medium = threepid["medium"].lower()
                        address = threepid["address"].lower()
                        external_3pids.append({"medium": medium, "address": address})
                        logger.info("Looking for 3PID %s:%s in user profile", medium, address)

                        validated_at = time_msec()
                        if not (yield store.get_user_id_by_threepid(medium, address)):
                            logger.info("3PID is not present, adding")
                            yield store.user_add_threepid(
                                user_id,
                                medium,
                                address,
                                validated_at,
                                validated_at
                            )
                        else:
                            logger.info("3PID is present, skipping")

                    if (self.config.replaceThreepid):
                        for threepid in (yield store.user_get_threepids(user_id)):
                            medium = threepid["medium"].lower()
                            address = threepid["address"].lower()
                            if {"medium": medium, "address": address} not in external_3pids:
                                logger.info("3PID is not present in external datastore, deleting")
                                yield store.user_delete_threepid(
                                    user_id,
                                    medium,
                                    address
                                )

            else:
                logger.info("3PIDs were not updated due to policy")
        else:
            logger.info("No profile data")

        defer.returnValue(True)

    @staticmethod
    def parse_config(config):
        # verify config sanity
        _require_keys(config, ["endpoint"])

        class _RestConfig(object):
            endpoint = ''
            regLower = True
            setNameOnRegister = True
            setNameOnLogin = False
            updateThreepid = True
            replaceThreepid = False

        rest_config = _RestConfig()
        rest_config.endpoint = config["endpoint"]

        try:
            rest_config.regLower = config['policy']['registration']['username']['enforceLowercase']
        except TypeError:
            # we don't care
            pass
        except KeyError:
            # we don't care
            pass

        try:
            rest_config.setNameOnRegister = config['policy']['registration']['profile']['name']
        except TypeError:
            # we don't care
            pass
        except KeyError:
            # we don't care
            pass

        try:
            rest_config.setNameOnLogin = config['policy']['login']['profile']['name']
        except TypeError:
            # we don't care
            pass
        except KeyError:
            # we don't care
            pass

        try:
            rest_config.updateThreepid = config['policy']['all']['threepid']['update']
        except TypeError:
            # we don't care
            pass
        except KeyError:
            # we don't care
            pass

        try:
            rest_config.replaceThreepid = config['policy']['all']['threepid']['replace']
        except TypeError:
            # we don't care
            pass
        except KeyError:
            # we don't care
            pass

        return rest_config


def _require_keys(config, required):
    missing = [key for key in required if key not in config]
    if missing:
        raise Exception(
            "REST Auth enabled but missing required config values: {}".format(
                ", ".join(missing)
            )
        )


def time_msec():
    """Get the current timestamp in milliseconds
    """
    return int(time.time() * 1000)
