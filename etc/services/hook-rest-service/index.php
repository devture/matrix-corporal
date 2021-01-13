<?php

$respondWithJsonAndExit = function (array $responsePayload): void {
	$responsePayloadString = json_encode($responsePayload);

	echo $responsePayloadString;

	file_put_contents('php://stdout', sprintf("Responding with: %s\n", $responsePayloadString));

	exit;
};

if ($_SERVER['REQUEST_URI'] === '/reject/with-33-percent-chance') {
	if (rand(0, 2) === 0) {
		$respondWithJsonAndExit([
			'id' => 'rejecting-unlucky-ones',
			'action' => 'reject',
			'responseStatusCode' => 403,
			'rejectionErrorCode' => 'M_FORBIDDEN',
			'rejectionErrorMessage' => 'Rejecting it via a hook delivered from a REST service',
		]);
	}

	$respondWithJsonAndExit([
		'id' => 'allowing-lucky-ones',
		'action' => 'pass.unmodified',
	]);
}

if ($_SERVER['REQUEST_URI'] === '/reject/forbidden') {
	$respondWithJsonAndExit([
		'id' => 'rejection-response-hook',
		'action' => 'reject',
		'responseStatusCode' => 403,
		'rejectionErrorCode' => 'M_FORBIDDEN',
		'rejectionErrorMessage' => 'Rejecting it via a hook delivered from a REST service',
	]);
}

if ($_SERVER['REQUEST_URI'] === '/inject-something-into-request') {
	$respondWithJsonAndExit([
		'id' => 'injection-request-hook',
		'action' => 'pass.modifiedRequest',
		"injectJSONIntoRequest" => [
			'customKey' => 'value',
		],
		'injectHeadersIntoRequest' => [
			'X-Custom-Header' => 'Header-Value',
		],
	]);
}

if ($_SERVER['REQUEST_URI'] === '/inject-something-into-response') {
	$respondWithJsonAndExit([
		'id' => 'injection-response-hook',
		'action' => 'pass.modifiedResponse',
		"injectJSONIntoResponse" => [
			'customKey' => 'value',
		],
		'injectHeadersIntoResponse' => [
			'X-Custom-Header' => 'Header-Value',
		],
	]);
}

if ($_SERVER['REQUEST_URI'] === '/respond-with-something') {
	// We could read the request (and possibly response) information here,
	// and act depending on that.
	//
	// See how we do it for the `/dump` handler for an example.
	$respondWithJsonAndExit([
		'id' => 'respond-directly',
		'action' => 'respond',
		"responseStatusCode" => 200,
		'responsePayload' => [
			'message' => 'This response is coming from the REST service',
		],
	]);
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

	$respondWithJsonAndExit([
		'id' => 'passed-after-dump',
		'action' => 'pass.unmodified',
	]);
}

$respondWithJsonAndExit([
	'id' => 'default',
	'action' => 'pass.unmodified',
]);
