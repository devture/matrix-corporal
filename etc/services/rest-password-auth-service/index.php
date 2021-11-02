<?php

$respondWithJsonAndExit = function (array $responsePayload): void {
	$responsePayloadString = json_encode($responsePayload);

	echo $responsePayloadString;

	file_put_contents('php://stdout', sprintf("Responding with: %s\n", $responsePayloadString));

	exit;
};

// This is an example implementation of the REST auth Password Provider.
// Learn more here: https://github.com/ma1uta/matrix-synapse-rest-password-provider
if ($_SERVER['REQUEST_METHOD'] === 'POST' && $_SERVER['REQUEST_URI'] === '/_matrix-internal/identity/v1/check_credentials') {
	$bodyPayload = file_get_contents('php://input');
	$payload = json_decode($bodyPayload, true, 3);
	if ($payload === null) {
		$respondWithJsonAndExit([
			'auth' => [
				'success' => false,
				'message' => 'Invalid JSON payload',
			],
		]);
	}

	if (!array_key_exists('user', $payload) || !array_key_exists('id', $payload['user']) || !array_key_exists('password', $payload['user'])) {
		$respondWithJsonAndExit([
			'auth' => [
				'success' => false,
				'message' => 'Invalid payload (no user field or id/password subfields)',
			],
		]);
	}

	list($matrixId, $password) = [$payload['user']['id'], $payload['user']['password']];

	// You need to validate `$matrixId` and `$password` here.
	// For this example, we simply authenticate anyone with any password.

	$respondWithJsonAndExit([
		'auth' => [
			'success' => true,
			'mxid' => $matrixId,
		]
	]);
}

$respondWithJsonAndExit([
	'auth' => [
		'success' => false,
		'message' => 'Bad call (incorrect route or HTTP request method)',
	],
]);
