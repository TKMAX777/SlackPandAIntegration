package main

var Settings struct {
	Panda struct {
		ECS_ID     string
		PASSWORD   string
		JSESSIONID string
	}
	Slack struct {
		Token      string
		EventToken string
	}
}
