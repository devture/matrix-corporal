<?php

if ($_SERVER['REQUEST_URI'] === '/reject/with-33-percent-chance') {
	if (rand(0, 2) === 0) {
		echo json_encode([
			'id' => 'rejecting-unlucky-ones',
			'action' => 'reject',
			'responseStatusCode' => 403,
			'rejectionErrorCode' => 'M_FORBIDDEN',
			'rejectionErrorMessage' => 'Rejecting it via a hook',
		]);
	} else {
		echo json_encode([
			'id' => 'allowing-lucky-ones',
			'action' => 'pass.unmodified',
		]);
	}

	exit;
}

if ($_SERVER['REQUEST_URI'] === '/reject/forbidden') {
	echo json_encode([
		'id' => 'rejection-response-hook',
		'action' => 'reject',
		'responseStatusCode' => 403,
		'rejectionErrorCode' => 'M_FORBIDDEN',
		'rejectionErrorMessage' => 'Rejecting it via a hook',
	]);

	exit;
}

if ($_SERVER['REQUEST_URI'] === '/inject-something') {
	echo json_encode([
		'id' => 'injection-response-hook',
		'action' => 'pass.injectJSONIntoResponse',
		"injectJSONIntoResponse" => [
			'customKey' => 'value',
		],
		'injectHeadersIntoResponse' => [
			'X-Custom-Header' => 'Header-Value',
		],
	]);

	exit;
}


if ($_SERVER['REQUEST_URI'] === '/dump') {
	$payload = file_get_contents('php://input');

	file_put_contents('php://stdout', sprintf("Request: %s\n", print_r($_SERVER, true)));
	file_put_contents('php://stdout', sprintf("Payload: %s\n", $payload));

	// The payload may or may not be JSON.
	// So errors below may mean malformed JSON input, or (in rare cases) something different than JSON.
	$json = json_decode($payload);
	if (json_last_error() != JSON_ERROR_NONE) {
		file_put_contents('php://stdout', sprintf(
			"Payload parsing error (%s): %s\n",
			json_last_error(),
			json_last_error_msg(),
		));
	} else {
		file_put_contents('php://stdout', "Payload JSON parsing: OK\n");
	}

	echo json_encode([
		'id' => 'passed-after-dump',
		'action' => 'pass.unmodified',
	]);

	exit;
}

echo json_encode([
	'id' => 'default',
	'action' => 'pass.unmodified',
]);
