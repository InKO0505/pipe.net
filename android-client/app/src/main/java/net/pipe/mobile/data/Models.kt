package net.pipe.mobile.data

data class ChatSummary(
    val id: String,
    val title: String,
    val subtitle: String,
    val unreadCount: Int = 0,
    val mentionCount: Int = 0,
    val isPrivate: Boolean = false,
)

data class MessageItem(
    val id: String,
    val channelId: String,
    val authorId: String,
    val author: String,
    val body: String,
    val timeLabel: String,
    val isMine: Boolean = false,
    val isEdited: Boolean = false,
    val replyPreview: String = "",
)

data class ProfileSummary(
    val username: String,
    val role: String,
    val bio: String,
    val endpoint: String,
)

data class SessionBundle(
    val token: String,
    val profile: ProfileSummary,
)

data class UserSummary(
    val id: String,
    val username: String,
    val role: String,
    val color: String,
    val bio: String,
)

data class ModLogItem(
    val actor: String,
    val target: String,
    val channel: String,
    val action: String,
    val details: String,
    val createdAt: String,
)
