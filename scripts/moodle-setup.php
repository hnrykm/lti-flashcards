<?php
// Registers the Flashcards tool in the local Moodle and creates a test course.
// Run through scripts/moodle-setup.sh; safe to run more than once.
// Prints one JSON line: {"client_id": ..., "deployment_id": ..., ...}

define('CLI_SCRIPT', true);
require('/var/www/html/config.php');
require_once($CFG->dirroot . '/mod/lti/locallib.php');
require_once($CFG->dirroot . '/course/lib.php');
require_once($CFG->dirroot . '/course/modlib.php');
require_once($CFG->dirroot . '/user/lib.php');
require_once($CFG->libdir . '/enrollib.php');

\core\session\manager::set_user(get_admin());
$tool = rtrim(getenv('TOOL_URL') ?: 'http://localhost:3000', '/');

// 1. The LTI 1.3 tool registration (Site administration > Plugins > External tool).
$type = $DB->get_record('lti_types', ['name' => 'Flashcards']);
if (!$type) {
    $t = new stdClass();
    $t->state = LTI_TOOL_STATE_CONFIGURED;
    $t->course = SITEID;

    $c = new stdClass();
    $c->lti_typename = 'Flashcards';
    $c->lti_description = 'LTI 1.3 flashcards tool';
    $c->lti_toolurl = $tool . '/lti/launch';
    $c->lti_ltiversion = LTI_VERSION_1P3;
    $c->lti_keytype = LTI_JWK_KEYSET;
    $c->lti_publickeyset = $tool . '/.well-known/jwks.json'; // unused: the tool never signs messages
    $c->lti_initiatelogin = $tool . '/lti/login';
    $c->lti_redirectionuris = $tool . '/lti/launch';
    $c->lti_coursevisible = LTI_COURSEVISIBLE_ACTIVITYCHOOSER;
    $c->lti_launchcontainer = LTI_LAUNCH_CONTAINER_EMBED_NO_BLOCKS;
    $c->lti_sendname = LTI_SETTING_ALWAYS;
    $c->lti_sendemailaddr = LTI_SETTING_ALWAYS;
    $c->lti_acceptgrades = LTI_SETTING_NEVER;

    $type = $DB->get_record('lti_types', ['id' => lti_add_type($t, $c)], '*', MUST_EXIST);
}

// 2. A course with a teacher (admin) and a student.
$course = $DB->get_record('course', ['shortname' => 'BIO101']);
if (!$course) {
    $course = create_course((object) [
        'fullname' => 'Biology 101',
        'shortname' => 'BIO101',
        'category' => core_course_category::get_default()->id,
    ]);
}

$student = $DB->get_record('user', ['username' => 'student']);
if (!$student) {
    $id = user_create_user((object) [
        'username' => 'student',
        'password' => 'Student123!',
        'firstname' => 'Sam',
        'lastname' => 'Student',
        'email' => 'student@example.com',
        'auth' => 'manual',
        'confirmed' => 1,
        'mnethostid' => $CFG->mnet_localhost_id,
    ]);
    $student = $DB->get_record('user', ['id' => $id]);
}

$roles = $DB->get_records_menu('role', null, '', 'shortname, id');
enrol_try_internal_enrol($course->id, get_admin()->id, $roles['editingteacher']);
enrol_try_internal_enrol($course->id, $student->id, $roles['student']);

// 3. The Flashcards activity in the course.
$lti = $DB->get_record('lti', ['course' => $course->id, 'typeid' => $type->id]);
if (!$lti) {
    $info = create_module((object) [
        'modulename' => 'lti',
        'course' => $course->id,
        'section' => 1,
        'visible' => 1,
        'name' => 'Flashcards',
        'introeditor' => ['text' => '', 'format' => FORMAT_HTML, 'itemid' => 0],
        'typeid' => $type->id,
        'toolurl' => '',
        'launchcontainer' => LTI_LAUNCH_CONTAINER_DEFAULT,
        'instructorchoicesendname' => 1,
        'instructorchoicesendemailaddr' => 1,
        'instructorchoiceacceptgrades' => 0,
        'grade' => 0,
    ]);
    $cmid = $info->coursemodule;
} else {
    $cmid = get_coursemodule_from_instance('lti', $lti->id, $course->id)->id;
}

echo json_encode([
    'issuer' => $CFG->wwwroot,
    'client_id' => $type->clientid,
    'deployment_id' => (string) $type->id,
    'auth_login_url' => $CFG->wwwroot . '/mod/lti/auth.php',
    'jwks_url' => $CFG->wwwroot . '/mod/lti/certs.php',
    'activity_url' => $CFG->wwwroot . '/mod/lti/view.php?id=' . $cmid,
]), "\n";
