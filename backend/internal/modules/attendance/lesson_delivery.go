package attendance

type LessonDeliveryPolicy struct {
	RequiresTopicWhenDone    bool
	RequiresReasonWhenMissed bool
}

var DefaultLessonDeliveryPolicy = LessonDeliveryPolicy{RequiresTopicWhenDone: true, RequiresReasonWhenMissed: true}
