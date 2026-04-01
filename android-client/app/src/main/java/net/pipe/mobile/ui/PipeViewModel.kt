package net.pipe.mobile.ui

import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import net.pipe.mobile.data.ChatRepository
import net.pipe.mobile.data.ChatSummary
import net.pipe.mobile.data.MessageItem
import net.pipe.mobile.data.ModLogItem
import net.pipe.mobile.data.ProfileSummary
import net.pipe.mobile.data.UserSummary

enum class SheetContent {
    None,
    Members,
    UserLookup,
    ModLog,
}

data class PipeUiState(
    val loading: Boolean = false,
    val endpoint: String = "http://10.0.2.2:8080",
    val usernameInput: String = "",
    val token: String? = null,
    val profile: ProfileSummary? = null,
    val chats: List<ChatSummary> = emptyList(),
    val selectedChatId: String? = null,
    val messages: List<MessageItem> = emptyList(),
    val showingMentions: Boolean = false,
    val selectedMessageId: String? = null,
    val replyingToMessageId: String? = null,
    val composerText: String = "",
    val editingMessageId: String? = null,
    val activeSheet: SheetContent = SheetContent.None,
    val members: List<UserSummary> = emptyList(),
    val modLog: List<ModLogItem> = emptyList(),
    val lookedUpUser: UserSummary? = null,
    val quickDmInput: String = "",
    val newChannelName: String = "",
    val newChannelPrivate: Boolean = false,
    val memberActionInput: String = "",
    val error: String? = null,
)

class PipeViewModel(
    private val repository: ChatRepository,
    private val prefs: AppPrefs,
) : ViewModel() {
    private var pollJob: Job? = null
    private val _uiState = MutableStateFlow(
        PipeUiState(
            endpoint = prefs.endpoint(),
            usernameInput = prefs.username(),
        ),
    )
    val uiState: StateFlow<PipeUiState> = _uiState.asStateFlow()

    fun updateEndpoint(value: String) {
        prefs.saveEndpoint(value)
        _uiState.value = _uiState.value.copy(endpoint = value)
    }

    fun updateUsername(value: String) {
        prefs.saveUsername(value)
        _uiState.value = _uiState.value.copy(usernameInput = value)
    }

    fun connect() {
        val endpoint = uiState.value.endpoint.trim()
        val username = uiState.value.usernameInput.trim()
        if (endpoint.isBlank() || username.isBlank()) {
            _uiState.value = _uiState.value.copy(error = "Endpoint and username are required.")
            return
        }

        viewModelScope.launch {
            _uiState.value = _uiState.value.copy(loading = true, error = null)
            runCatching {
                val session = repository.login(endpoint, username)
                prefs.saveEndpoint(endpoint)
                prefs.saveUsername(session.profile.username)
                val chats = repository.loadChats(endpoint, session.token)
                val selectedChatId = chats.firstOrNull()?.id
                val messages = selectedChatId?.let { repository.loadMessages(endpoint, session.token, it) }.orEmpty()
                    .markMine(session.profile.username)
                PipeUiState(
                    loading = false,
                    endpoint = endpoint,
                    usernameInput = session.profile.username,
                    token = session.token,
                    profile = session.profile,
                    chats = chats,
                    selectedChatId = selectedChatId,
                    messages = messages,
                    showingMentions = false,
                    quickDmInput = "",
                    memberActionInput = "",
                )
            }.onSuccess {
                _uiState.value = it
                startPolling()
            }.onFailure {
                _uiState.value = _uiState.value.copy(loading = false, error = it.message ?: "Connection failed.")
            }
        }
    }

    fun refresh() {
        val state = uiState.value
        val token = state.token ?: return
        val profile = state.profile ?: return
        viewModelScope.launch {
            _uiState.value = state.copy(loading = true, error = null)
            runCatching {
                val refreshedProfile = repository.loadProfile(state.endpoint, token)
                val chats = repository.loadChats(state.endpoint, token)
                val selectedChatId = state.selectedChatId ?: chats.firstOrNull()?.id
                val messages = if (state.showingMentions) {
                    repository.loadMentions(state.endpoint, token)
                } else {
                    selectedChatId?.let { repository.loadMessages(state.endpoint, token, it) }.orEmpty()
                }.markMine(refreshedProfile.username)
                state.copy(
                    loading = false,
                    profile = refreshedProfile,
                    chats = chats,
                    selectedChatId = selectedChatId,
                    messages = messages,
                )
            }.onSuccess {
                _uiState.value = it
            }.onFailure {
                _uiState.value = state.copy(loading = false, error = it.message ?: "Refresh failed.")
            }
        }
    }

    fun selectChat(chatId: String) {
        val state = uiState.value
        val token = state.token ?: return
        val profile = state.profile ?: return
        viewModelScope.launch {
            _uiState.value = state.copy(loading = true, selectedChatId = chatId, error = null)
            runCatching {
                repository.loadMessages(state.endpoint, token, chatId).markMine(profile.username)
            }.onSuccess { messages ->
                _uiState.value = state.copy(
                    loading = false,
                    selectedChatId = chatId,
                    messages = messages,
                    showingMentions = false,
                    selectedMessageId = null,
                    replyingToMessageId = null,
                    editingMessageId = null,
                    composerText = "",
                )
            }.onFailure {
                _uiState.value = state.copy(loading = false, error = it.message ?: "Loading messages failed.")
            }
        }
    }

    fun updateComposerText(value: String) {
        _uiState.value = _uiState.value.copy(composerText = value)
    }

    fun selectMessage(messageId: String) {
        val state = uiState.value
        _uiState.value = state.copy(
            selectedMessageId = if (state.selectedMessageId == messageId) null else messageId,
        )
    }

    fun startEditingSelectedMessage() {
        val state = uiState.value
        val selected = state.messages.firstOrNull { it.id == state.selectedMessageId } ?: return
        if (!selected.isMine) return
        _uiState.value = state.copy(
            editingMessageId = selected.id,
            replyingToMessageId = null,
            composerText = selected.body,
        )
    }

    fun startReplyToSelectedMessage() {
        val state = uiState.value
        val selected = state.messages.firstOrNull { it.id == state.selectedMessageId } ?: return
        _uiState.value = state.copy(
            replyingToMessageId = selected.id,
            editingMessageId = null,
            selectedMessageId = null,
        )
    }

    fun cancelEditing() {
        _uiState.value = _uiState.value.copy(
            editingMessageId = null,
            replyingToMessageId = null,
            selectedMessageId = null,
            composerText = "",
        )
    }

    fun submitComposer() {
        val state = uiState.value
        val token = state.token ?: return
        val chatId = state.selectedChatId ?: return
        val profile = state.profile ?: return
        val content = state.composerText
        if (content.isBlank()) {
            return
        }

        viewModelScope.launch {
            if (state.editingMessageId != null) {
                runCatching {
                    repository.editMessage(state.endpoint, token, chatId, state.editingMessageId, content)
                }.onSuccess { message ->
                    _uiState.value = state.copy(
                        messages = state.messages.map {
                            if (it.id == message.id) message.copy(isMine = message.author == profile.username) else it
                        },
                        composerText = "",
                        editingMessageId = null,
                        selectedMessageId = null,
                        error = null,
                    )
                }.onFailure {
                    _uiState.value = state.copy(error = it.message ?: "Edit failed.")
                }
            } else {
                runCatching {
                    repository.sendMessage(
                        endpoint = state.endpoint,
                        token = token,
                        chatId = chatId,
                        content = content,
                        replyToId = state.replyingToMessageId,
                    )
                }.onSuccess { message ->
                    _uiState.value = state.copy(
                        messages = state.messages + message.copy(isMine = message.author == profile.username),
                        composerText = "",
                        replyingToMessageId = null,
                        selectedMessageId = null,
                        error = null,
                    )
                }.onFailure {
                    _uiState.value = state.copy(error = it.message ?: "Send failed.")
                }
            }
        }
    }

    fun deleteSelectedMessage() {
        val state = uiState.value
        val token = state.token ?: return
        val chatId = state.selectedChatId ?: return
        val selected = state.messages.firstOrNull { it.id == state.selectedMessageId } ?: return
        if (!selected.isMine) return
        viewModelScope.launch {
            runCatching {
                repository.deleteMessage(state.endpoint, token, chatId, selected.id)
            }.onSuccess {
                _uiState.value = state.copy(
                    messages = state.messages.map {
                        if (it.id == selected.id) it.copy(body = "[deleted]", isEdited = true) else it
                    },
                    selectedMessageId = null,
                    replyingToMessageId = null,
                    editingMessageId = null,
                    composerText = "",
                    error = null,
                )
            }.onFailure {
                _uiState.value = state.copy(error = it.message ?: "Delete failed.")
            }
        }
    }

    fun showMentions() {
        val state = uiState.value
        val token = state.token ?: return
        val profile = state.profile ?: return
        viewModelScope.launch {
            _uiState.value = state.copy(loading = true, error = null)
            runCatching {
                repository.loadMentions(state.endpoint, token).markMine(profile.username)
            }.onSuccess { mentions ->
                _uiState.value = state.copy(
                    loading = false,
                    messages = mentions,
                    showingMentions = true,
                    selectedMessageId = null,
                    replyingToMessageId = null,
                    editingMessageId = null,
                )
            }.onFailure {
                _uiState.value = state.copy(loading = false, error = it.message ?: "Failed to load mentions.")
            }
        }
    }

    fun updateQuickDmInput(value: String) {
        _uiState.value = _uiState.value.copy(quickDmInput = value)
    }

    fun updateNewChannelName(value: String) {
        _uiState.value = _uiState.value.copy(newChannelName = value)
    }

    fun toggleNewChannelPrivacy() {
        _uiState.value = _uiState.value.copy(newChannelPrivate = !_uiState.value.newChannelPrivate)
    }

    fun updateMemberActionInput(value: String) {
        _uiState.value = _uiState.value.copy(memberActionInput = value)
    }

    fun openMembers() {
        val state = uiState.value
        val token = state.token ?: return
        val chatId = state.selectedChatId ?: return
        viewModelScope.launch {
            runCatching {
                repository.loadMembers(state.endpoint, token, chatId)
            }.onSuccess { members ->
                _uiState.value = state.copy(activeSheet = SheetContent.Members, members = members, error = null)
            }.onFailure {
                _uiState.value = state.copy(error = it.message ?: "Failed to load members.")
            }
        }
    }

    fun openModLog() {
        val state = uiState.value
        val token = state.token ?: return
        viewModelScope.launch {
            runCatching {
                repository.loadModLog(state.endpoint, token)
            }.onSuccess { logs ->
                _uiState.value = state.copy(activeSheet = SheetContent.ModLog, modLog = logs, error = null)
            }.onFailure {
                _uiState.value = state.copy(error = it.message ?: "Failed to load moderation log.")
            }
        }
    }

    fun lookupUser(username: String) {
        val state = uiState.value
        val token = state.token ?: return
        if (username.isBlank()) return
        viewModelScope.launch {
            runCatching {
                repository.loadUser(state.endpoint, token, username)
            }.onSuccess { user ->
                _uiState.value = state.copy(activeSheet = SheetContent.UserLookup, lookedUpUser = user, error = null)
            }.onFailure {
                _uiState.value = state.copy(error = it.message ?: "Failed to load user.")
            }
        }
    }

    fun createDirectMessage() {
        val state = uiState.value
        val token = state.token ?: return
        val username = state.quickDmInput.trim()
        if (username.isBlank()) return
        viewModelScope.launch {
            runCatching {
                repository.createDm(state.endpoint, token, username)
            }.onSuccess { chat ->
                val chats = listOf(chat) + state.chats.filterNot { it.id == chat.id }
                _uiState.value = state.copy(
                    chats = chats,
                    quickDmInput = "",
                    error = null,
                )
                selectChat(chat.id)
            }.onFailure {
                _uiState.value = state.copy(error = it.message ?: "Failed to create DM.")
            }
        }
    }

    fun createChannel() {
        val state = uiState.value
        val token = state.token ?: return
        val name = state.newChannelName.trim()
        if (name.isBlank()) return
        viewModelScope.launch {
            runCatching {
                repository.createChannel(state.endpoint, token, name, state.newChannelPrivate)
            }.onSuccess { chat ->
                _uiState.value = state.copy(
                    chats = listOf(chat) + state.chats.filterNot { it.id == chat.id },
                    newChannelName = "",
                    newChannelPrivate = false,
                    error = null,
                )
                selectChat(chat.id)
            }.onFailure {
                _uiState.value = state.copy(error = it.message ?: "Channel creation failed.")
            }
        }
    }

    fun inviteMember() {
        updateMembers(add = true)
    }

    fun removeMember() {
        updateMembers(add = false)
    }

    fun closeSheet() {
        _uiState.value = _uiState.value.copy(activeSheet = SheetContent.None)
    }

    private fun updateMembers(add: Boolean) {
        val state = uiState.value
        val token = state.token ?: return
        val chatId = state.selectedChatId ?: return
        val username = state.memberActionInput.trim()
        if (username.isBlank()) return
        viewModelScope.launch {
            runCatching {
                if (add) {
                    repository.inviteMember(state.endpoint, token, chatId, username)
                } else {
                    repository.removeMember(state.endpoint, token, chatId, username)
                }
                repository.loadMembers(state.endpoint, token, chatId)
            }.onSuccess { members ->
                _uiState.value = state.copy(
                    activeSheet = SheetContent.Members,
                    members = members,
                    memberActionInput = "",
                    error = null,
                )
            }.onFailure {
                _uiState.value = state.copy(
                    error = it.message ?: if (add) "Invite failed." else "Remove failed.",
                )
            }
        }
    }

    private fun startPolling() {
        pollJob?.cancel()
        pollJob = viewModelScope.launch {
            while (true) {
                delay(5000)
                val state = uiState.value
                if (
                    state.token != null &&
                    !state.loading &&
                    state.editingMessageId == null &&
                    state.composerText.isBlank()
                ) {
                    refresh()
                }
            }
        }
    }

    companion object {
        fun factory(repository: ChatRepository, prefs: AppPrefs): ViewModelProvider.Factory {
            return object : ViewModelProvider.Factory {
                @Suppress("UNCHECKED_CAST")
                override fun <T : ViewModel> create(modelClass: Class<T>): T {
                    return PipeViewModel(repository, prefs) as T
                }
            }
        }
    }

    override fun onCleared() {
        pollJob?.cancel()
        super.onCleared()
    }
}

private fun List<MessageItem>.markMine(username: String): List<MessageItem> {
    return map { it.copy(isMine = it.author == username) }
}
