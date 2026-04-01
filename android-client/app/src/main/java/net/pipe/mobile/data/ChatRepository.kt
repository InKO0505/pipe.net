package net.pipe.mobile.data

import java.io.BufferedReader
import java.io.OutputStreamWriter
import java.net.HttpURLConnection
import java.net.URL
import org.json.JSONArray
import org.json.JSONObject

interface ChatRepository {
    suspend fun login(endpoint: String, username: String): SessionBundle
    suspend fun loadProfile(endpoint: String, token: String): ProfileSummary
    suspend fun loadChats(endpoint: String, token: String): List<ChatSummary>
    suspend fun loadMessages(endpoint: String, token: String, chatId: String): List<MessageItem>
    suspend fun loadMentions(endpoint: String, token: String): List<MessageItem>
    suspend fun loadMembers(endpoint: String, token: String, chatId: String): List<UserSummary>
    suspend fun loadUser(endpoint: String, token: String, username: String): UserSummary
    suspend fun createChannel(endpoint: String, token: String, name: String, isPrivate: Boolean): ChatSummary
    suspend fun createDm(endpoint: String, token: String, username: String): ChatSummary
    suspend fun inviteMember(endpoint: String, token: String, chatId: String, username: String)
    suspend fun removeMember(endpoint: String, token: String, chatId: String, username: String)
    suspend fun loadModLog(endpoint: String, token: String): List<ModLogItem>
    suspend fun editMessage(endpoint: String, token: String, chatId: String, messageId: String, content: String): MessageItem
    suspend fun deleteMessage(endpoint: String, token: String, chatId: String, messageId: String)
    suspend fun sendMessage(
        endpoint: String,
        token: String,
        chatId: String,
        content: String,
        replyToId: String? = null,
    ): MessageItem
}

class MobileApiRepository : ChatRepository {
    override suspend fun login(endpoint: String, username: String): SessionBundle {
        val response = request(
            endpoint = endpoint,
            path = "/api/mobile/login",
            method = "POST",
            body = JSONObject()
                .put("username", username.trim())
                .toString(),
        )
        val token = response.getString("token")
        val user = response.getJSONObject("user")
        return SessionBundle(
            token = token,
            profile = ProfileSummary(
                username = user.getString("username"),
                role = user.getString("role"),
                bio = user.optString("bio"),
                endpoint = endpoint,
            ),
        )
    }

    override suspend fun loadProfile(endpoint: String, token: String): ProfileSummary {
        val response = request(endpoint, "/api/mobile/me", "GET", token = token)
        val user = response.getJSONObject("user")
        return ProfileSummary(
            username = user.getString("username"),
            role = user.getString("role"),
            bio = user.optString("bio"),
            endpoint = endpoint,
        )
    }

    override suspend fun loadChats(endpoint: String, token: String): List<ChatSummary> {
        val response = request(endpoint, "/api/mobile/channels", "GET", token = token)
        val items = response.getJSONArray("channels")
        return buildList {
            for (index in 0 until items.length()) {
                val chat = items.getJSONObject(index)
                add(
                    ChatSummary(
                        id = chat.getString("id"),
                        title = chat.getString("name"),
                        subtitle = chat.optString("topic").ifBlank { chat.optString("kind") },
                        unreadCount = chat.optInt("unread_count"),
                        mentionCount = chat.optInt("mention_count"),
                        isPrivate = chat.optBoolean("is_private"),
                    ),
                )
            }
        }
    }

    override suspend fun loadMessages(endpoint: String, token: String, chatId: String): List<MessageItem> {
        val response = request(endpoint, "/api/mobile/channels/$chatId/messages", "GET", token = token)
        val items = response.getJSONArray("messages")
        return buildList {
            for (index in 0 until items.length()) {
                add(items.getJSONObject(index).toMessageItem())
            }
        }
    }

    override suspend fun loadMentions(endpoint: String, token: String): List<MessageItem> {
        val response = request(endpoint, "/api/mobile/mentions", "GET", token = token)
        val items = response.getJSONArray("mentions")
        return buildList {
            for (index in 0 until items.length()) {
                add(items.getJSONObject(index).toMessageItem())
            }
        }
    }

    override suspend fun loadMembers(endpoint: String, token: String, chatId: String): List<UserSummary> {
        val response = request(endpoint, "/api/mobile/channels/$chatId/members", "GET", token = token)
        val items = response.getJSONArray("members")
        return buildList {
            for (index in 0 until items.length()) {
                add(items.getJSONObject(index).toUserSummary())
            }
        }
    }

    override suspend fun loadUser(endpoint: String, token: String, username: String): UserSummary {
        val response = request(endpoint, "/api/mobile/users/${username.trim()}", "GET", token = token)
        return response.getJSONObject("user").toUserSummary()
    }

    override suspend fun createChannel(endpoint: String, token: String, name: String, isPrivate: Boolean): ChatSummary {
        val response = request(
            endpoint = endpoint,
            path = "/api/mobile/channels",
            method = "POST",
            token = token,
            body = JSONObject()
                .put("name", name.trim())
                .put("is_private", isPrivate)
                .toString(),
        )
        return response.getJSONObject("channel").toChatSummary()
    }

    override suspend fun createDm(endpoint: String, token: String, username: String): ChatSummary {
        val response = request(
            endpoint = endpoint,
            path = "/api/mobile/dm",
            method = "POST",
            token = token,
            body = JSONObject().put("username", username.trim()).toString(),
        )
        return response.getJSONObject("channel").toChatSummary()
    }

    override suspend fun inviteMember(endpoint: String, token: String, chatId: String, username: String) {
        request(
            endpoint = endpoint,
            path = "/api/mobile/channels/$chatId/invite",
            method = "POST",
            token = token,
            body = JSONObject().put("username", username.trim()).toString(),
        )
    }

    override suspend fun removeMember(endpoint: String, token: String, chatId: String, username: String) {
        request(
            endpoint = endpoint,
            path = "/api/mobile/channels/$chatId/remove",
            method = "POST",
            token = token,
            body = JSONObject().put("username", username.trim()).toString(),
        )
    }

    override suspend fun loadModLog(endpoint: String, token: String): List<ModLogItem> {
        val response = request(endpoint, "/api/mobile/modlog?limit=30", "GET", token = token)
        val items = response.getJSONArray("logs")
        return buildList {
            for (index in 0 until items.length()) {
                val item = items.getJSONObject(index)
                add(
                    ModLogItem(
                        actor = item.optString("ActorUsername"),
                        target = item.optString("TargetUsername"),
                        channel = item.optString("ChannelName"),
                        action = item.optString("Action"),
                        details = item.optString("Details"),
                        createdAt = item.optString("CreatedAt"),
                    ),
                )
            }
        }
    }

    override suspend fun editMessage(endpoint: String, token: String, chatId: String, messageId: String, content: String): MessageItem {
        val response = request(
            endpoint = endpoint,
            path = "/api/mobile/messages/$messageId?channel_id=$chatId",
            method = "PATCH",
            token = token,
            body = JSONObject().put("content", content).toString(),
        )
        return response.getJSONObject("message").toMessageItem()
    }

    override suspend fun deleteMessage(endpoint: String, token: String, chatId: String, messageId: String) {
        request(
            endpoint = endpoint,
            path = "/api/mobile/messages/$messageId?channel_id=$chatId",
            method = "DELETE",
            token = token,
        )
    }

    override suspend fun sendMessage(
        endpoint: String,
        token: String,
        chatId: String,
        content: String,
        replyToId: String?,
    ): MessageItem {
        val payload = JSONObject().put("content", content)
        if (!replyToId.isNullOrBlank()) {
            payload.put("reply_to_id", replyToId)
        }
        val response = request(
            endpoint = endpoint,
            path = "/api/mobile/channels/$chatId/messages",
            method = "POST",
            token = token,
            body = payload.toString(),
        )
        return response.getJSONObject("message").toMessageItem()
    }

    private fun request(
        endpoint: String,
        path: String,
        method: String,
        token: String? = null,
        body: String? = null,
    ): JSONObject {
        val base = endpoint.trim().removeSuffix("/")
        val url = URL(base + path)
        val connection = (url.openConnection() as HttpURLConnection).apply {
            requestMethod = method
            connectTimeout = 10_000
            readTimeout = 10_000
            setRequestProperty("Content-Type", "application/json; charset=utf-8")
            setRequestProperty("Accept", "application/json")
            if (!token.isNullOrBlank()) {
                setRequestProperty("Authorization", "Bearer $token")
            }
            doInput = true
            if (body != null) {
                doOutput = true
            }
        }

        if (body != null) {
            OutputStreamWriter(connection.outputStream).use { it.write(body) }
        }

        val responseCode = connection.responseCode
        val stream = if (responseCode in 200..299) connection.inputStream else connection.errorStream
        val text = stream?.bufferedReader()?.use(BufferedReader::readText).orEmpty()
        if (responseCode !in 200..299) {
            val message = runCatching { JSONObject(text).optString("error") }.getOrNull().orEmpty()
            throw IllegalStateException(message.ifBlank { "HTTP $responseCode" })
        }
        return JSONObject(text)
    }
}

private fun JSONObject.toChatSummary(): ChatSummary {
    return ChatSummary(
        id = getString("id"),
        title = getString("name"),
        subtitle = optString("topic").ifBlank { optString("kind") },
        unreadCount = optInt("unread_count"),
        mentionCount = optInt("mention_count"),
        isPrivate = optBoolean("is_private"),
    )
}

private fun JSONObject.toMessageItem(): MessageItem {
    return MessageItem(
        id = getString("id"),
        channelId = getString("channel_id"),
        authorId = getString("author_id"),
        author = getString("author_name"),
        body = getString("body"),
        timeLabel = getString("created_at").replace("T", " ").take(16),
        isMine = false,
        isEdited = optBoolean("is_edited"),
        replyPreview = buildString {
            val replyUser = optString("reply_to_username")
            val replyContent = optString("reply_to_content")
            if (replyUser.isNotBlank() || replyContent.isNotBlank()) {
                append(replyUser)
                if (replyUser.isNotBlank() && replyContent.isNotBlank()) {
                    append(": ")
                }
                append(replyContent)
            }
        },
    )
}

private fun JSONObject.toUserSummary(): UserSummary {
    return UserSummary(
        id = getString("id"),
        username = getString("username"),
        role = getString("role"),
        color = optString("color"),
        bio = optString("bio"),
    )
}
