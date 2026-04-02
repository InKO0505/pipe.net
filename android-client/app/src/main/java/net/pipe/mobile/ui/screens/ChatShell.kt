package net.pipe.mobile.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import net.pipe.mobile.data.ChatSummary
import net.pipe.mobile.data.MessageItem
import net.pipe.mobile.data.ModLogItem
import net.pipe.mobile.data.UserSummary
import net.pipe.mobile.ui.PipeUiState
import net.pipe.mobile.ui.PipeViewModel
import net.pipe.mobile.ui.SheetContent

@Composable
fun ChatShell(viewModel: PipeViewModel) {
    val state by viewModel.uiState.collectAsState()

    Surface(
        modifier = Modifier.fillMaxSize(),
        color = Color(0xFFF3EEE6),
    ) {
        if (state.token == null) {
            ConnectScreen(
                state = state,
                onEndpointChange = viewModel::updateEndpoint,
                onUsernameChange = viewModel::updateUsername,
                onConnect = viewModel::connect,
            )
        } else {
            ChatScreen(
                state = state,
                onDraftChange = viewModel::updateComposerText,
                onRefresh = viewModel::refresh,
                onShowMentions = viewModel::showMentions,
                onOpenMembers = viewModel::openMembers,
                onOpenModLog = viewModel::openModLog,
                onDmInputChange = viewModel::updateQuickDmInput,
                onCreateDm = viewModel::createDirectMessage,
                onLookupUser = viewModel::lookupUser,
                onCloseSheet = viewModel::closeSheet,
                onChatClick = viewModel::selectChat,
                onSelectMessage = viewModel::selectMessage,
                onStartReply = viewModel::startReplyToSelectedMessage,
                onStartEdit = viewModel::startEditingSelectedMessage,
                onCancelEdit = viewModel::cancelEditing,
                onDeleteSelected = viewModel::deleteSelectedMessage,
                onSend = viewModel::submitComposer,
                onNewChannelNameChange = viewModel::updateNewChannelName,
                onToggleNewChannelPrivacy = viewModel::toggleNewChannelPrivacy,
                onCreateChannel = viewModel::createChannel,
                onMemberInputChange = viewModel::updateMemberActionInput,
                onInviteMember = viewModel::inviteMember,
                onRemoveMember = viewModel::removeMember,
            )
        }
    }
}

@Composable
private fun ConnectScreen(
    state: PipeUiState,
    onEndpointChange: (String) -> Unit,
    onUsernameChange: (String) -> Unit,
    onConnect: () -> Unit,
) {
    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(
                Brush.linearGradient(
                    listOf(Color(0xFF132238), Color(0xFF825336), Color(0xFFF3EEE6)),
                ),
            )
            .padding(24.dp),
        contentAlignment = Alignment.Center,
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .clip(RoundedCornerShape(30.dp))
                .background(Color(0xFFF8F4EE))
                .padding(20.dp),
        ) {
            Text(
                text = "Pipe Net Mobile",
                style = MaterialTheme.typography.headlineMedium,
                fontWeight = FontWeight.Black,
                color = Color(0xFF111827),
            )
            Spacer(modifier = Modifier.height(8.dp))
            Text(
                text = "Open the app, save the server once, and use it like a regular messenger.",
                color = Color(0xFF475467),
            )
            Spacer(modifier = Modifier.height(20.dp))
            OutlinedTextField(
                value = state.endpoint,
                onValueChange = onEndpointChange,
                modifier = Modifier.fillMaxWidth(),
                label = { Text("Server address") },
                placeholder = { Text("http://10.0.2.2:8080") },
                singleLine = true,
                colors = pipeTextFieldColors(),
            )
            Spacer(modifier = Modifier.height(12.dp))
            OutlinedTextField(
                value = state.usernameInput,
                onValueChange = onUsernameChange,
                modifier = Modifier.fillMaxWidth(),
                label = { Text("Login") },
                placeholder = { Text("inko") },
                singleLine = true,
                colors = pipeTextFieldColors(),
            )
            if (state.error != null) {
                Spacer(modifier = Modifier.height(12.dp))
                Text(
                    text = state.error,
                    color = Color(0xFFB42318),
                    style = MaterialTheme.typography.bodyMedium,
                )
            }
            Spacer(modifier = Modifier.height(20.dp))
            Button(
                onClick = onConnect,
                modifier = Modifier.fillMaxWidth(),
                enabled = !state.loading,
            ) {
                if (state.loading) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(18.dp),
                        strokeWidth = 2.dp,
                        color = Color.White,
                    )
                } else {
                    Text("Connect")
                }
            }
            Spacer(modifier = Modifier.height(10.dp))
            Text(
                text = "Emulator: http://10.0.2.2:8080. Real phone: use http://<LAN-IP>:8080. HTTPS works only if the server has a valid TLS certificate.",
                color = Color(0xFF667085),
                style = MaterialTheme.typography.bodySmall,
            )
        }
    }
}

@Composable
private fun ChatScreen(
    state: PipeUiState,
    onDraftChange: (String) -> Unit,
    onRefresh: () -> Unit,
    onShowMentions: () -> Unit,
    onOpenMembers: () -> Unit,
    onOpenModLog: () -> Unit,
    onDmInputChange: (String) -> Unit,
    onCreateDm: () -> Unit,
    onLookupUser: (String) -> Unit,
    onCloseSheet: () -> Unit,
    onChatClick: (String) -> Unit,
    onSelectMessage: (String) -> Unit,
    onStartReply: () -> Unit,
    onStartEdit: () -> Unit,
    onCancelEdit: () -> Unit,
    onDeleteSelected: () -> Unit,
    onSend: () -> Unit,
    onNewChannelNameChange: (String) -> Unit,
    onToggleNewChannelPrivacy: () -> Unit,
    onCreateChannel: () -> Unit,
    onMemberInputChange: (String) -> Unit,
    onInviteMember: () -> Unit,
    onRemoveMember: () -> Unit,
) {
    val selectedMessage = state.messages.firstOrNull { it.id == state.selectedMessageId }
    val replyTarget = state.messages.firstOrNull { it.id == state.replyingToMessageId }
    val canManageMembers = state.profile?.role == "owner" || state.profile?.role == "admin"

    Column(
        modifier = Modifier
            .fillMaxSize()
            .navigationBarsPadding(),
    ) {
        Header(state = state, onShowMentions = onShowMentions, onRefresh = onRefresh)
        ProfileStrip(state = state)

        if (state.profile?.role == "owner") {
            ChannelComposerCard(
                state = state,
                onNameChange = onNewChannelNameChange,
                onTogglePrivacy = onToggleNewChannelPrivacy,
                onCreate = onCreateChannel,
            )
        }

        QuickActions(
            state = state,
            onDmInputChange = onDmInputChange,
            onCreateDm = onCreateDm,
            onOpenMembers = onOpenMembers,
            onOpenModLog = onOpenModLog,
            onLookupUser = onLookupUser,
        )

        if (state.showingMentions) {
            SectionLabel(text = "Inbox")
        } else {
            SectionLabel(text = "Chats")
        }

        LazyRow(
            modifier = Modifier
                .fillMaxWidth()
                .padding(vertical = 6.dp),
            horizontalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            items(state.chats, key = { it.id }) { chat ->
                ChatCard(
                    chat = chat,
                    selected = chat.id == state.selectedChatId,
                    onClick = { onChatClick(chat.id) },
                )
            }
        }

        MessageTimeline(
            messages = state.messages,
            selectedMessageId = state.selectedMessageId,
            onSelectMessage = onSelectMessage,
            modifier = Modifier
                .weight(1f)
                .padding(horizontal = 12.dp),
        )

        if (state.activeSheet != SheetContent.None) {
            InfoSheet(
                state = state,
                canManageMembers = canManageMembers,
                onMemberInputChange = onMemberInputChange,
                onInviteMember = onInviteMember,
                onRemoveMember = onRemoveMember,
                onClose = onCloseSheet,
            )
        }

        Column(
            modifier = Modifier
                .fillMaxWidth()
                .background(Color(0xFFF8F3EC))
                .padding(12.dp),
        ) {
            if (selectedMessage != null) {
                SelectedMessageBar(
                    selectedMessage = selectedMessage,
                    isEditing = state.editingMessageId != null,
                    onReply = onStartReply,
                    onEdit = onStartEdit,
                    onDelete = onDeleteSelected,
                    onCancel = onCancelEdit,
                )
                Spacer(modifier = Modifier.height(8.dp))
            }
            if (replyTarget != null || state.editingMessageId != null) {
                ReplyBanner(
                    replyTarget = replyTarget,
                    isEditing = state.editingMessageId != null,
                    onCancel = onCancelEdit,
                )
                Spacer(modifier = Modifier.height(8.dp))
            }
            if (state.error != null) {
                Text(
                    text = state.error,
                    color = Color(0xFFB42318),
                    style = MaterialTheme.typography.bodySmall,
                )
                Spacer(modifier = Modifier.height(8.dp))
            }
            OutlinedTextField(
                value = state.composerText,
                onValueChange = onDraftChange,
                modifier = Modifier.fillMaxWidth(),
                label = { Text(if (state.editingMessageId != null) "Edit message" else "Message") },
                placeholder = { Text(if (state.editingMessageId != null) "Update the selected message" else "Write something") },
                shape = RoundedCornerShape(24.dp),
                colors = pipeTextFieldColors(),
            )
            Spacer(modifier = Modifier.height(10.dp))
            Button(
                onClick = onSend,
                enabled = !state.loading && state.composerText.isNotBlank() && state.selectedChatId != null,
                modifier = Modifier.fillMaxWidth(),
            ) {
                if (state.loading) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(18.dp),
                        strokeWidth = 2.dp,
                        color = Color.White,
                    )
                } else {
                    Text(if (state.editingMessageId != null) "Save message" else "Send message")
                }
            }
        }
    }
}

@Composable
private fun Header(
    state: PipeUiState,
    onShowMentions: () -> Unit,
    onRefresh: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(Color(0xFF162232))
            .padding(horizontal = 16.dp, vertical = 14.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = state.profile?.username ?: "unknown",
                color = Color.White,
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Black,
            )
            Text(
                text = state.profile?.endpoint.orEmpty(),
                color = Color(0xFFD0D5DD),
                style = MaterialTheme.typography.bodySmall,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
        Spacer(modifier = Modifier.width(12.dp))
        TextButton(onClick = onShowMentions) {
            Text("Inbox", color = Color.White)
        }
        TextButton(onClick = onRefresh) {
            Text("Sync", color = Color.White)
        }
    }
}

@Composable
private fun ProfileStrip(state: PipeUiState) {
    val profile = state.profile ?: return
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 12.dp, vertical = 10.dp)
            .clip(RoundedCornerShape(26.dp))
            .background(Color(0xFFE7DACA))
            .padding(14.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            modifier = Modifier
                .size(46.dp)
                .clip(CircleShape)
                .background(Color(0xFF162232)),
            contentAlignment = Alignment.Center,
        ) {
            Text(
                text = profile.username.take(1).uppercase(),
                color = Color.White,
                fontWeight = FontWeight.Black,
            )
        }
        Spacer(modifier = Modifier.width(12.dp))
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = profile.role.uppercase(),
                color = Color(0xFF7C2D12),
                style = MaterialTheme.typography.labelMedium,
                fontWeight = FontWeight.Bold,
            )
            Text(
                text = profile.bio.ifBlank { "Ready to chat." },
                color = Color(0xFF344054),
                style = MaterialTheme.typography.bodyMedium,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
        }
    }
}

@Composable
private fun ChannelComposerCard(
    state: PipeUiState,
    onNameChange: (String) -> Unit,
    onTogglePrivacy: () -> Unit,
    onCreate: () -> Unit,
) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 12.dp)
            .clip(RoundedCornerShape(24.dp))
            .background(Color(0xFFFCFAF7))
            .padding(12.dp),
    ) {
        Text(
            text = "Create channel",
            color = Color(0xFF101828),
            fontWeight = FontWeight.Bold,
        )
        Spacer(modifier = Modifier.height(8.dp))
        OutlinedTextField(
            value = state.newChannelName,
            onValueChange = onNameChange,
            modifier = Modifier.fillMaxWidth(),
            label = { Text("Channel name") },
            singleLine = true,
            colors = pipeTextFieldColors(),
        )
        Spacer(modifier = Modifier.height(8.dp))
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = if (state.newChannelPrivate) "Private channel" else "Public channel",
                color = Color(0xFF475467),
            )
            Row {
                TextButton(onClick = onTogglePrivacy) {
                    Text(if (state.newChannelPrivate) "Make public" else "Make private")
                }
                Button(
                    onClick = onCreate,
                    enabled = state.newChannelName.isNotBlank() && !state.loading,
                ) {
                    Text("Create")
                }
            }
        }
    }
}

@Composable
private fun QuickActions(
    state: PipeUiState,
    onDmInputChange: (String) -> Unit,
    onCreateDm: () -> Unit,
    onOpenMembers: () -> Unit,
    onOpenModLog: () -> Unit,
    onLookupUser: (String) -> Unit,
) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 12.dp)
            .clip(RoundedCornerShape(24.dp))
            .background(Color(0xFFFCFAF7))
            .padding(12.dp),
    ) {
        OutlinedTextField(
            value = state.quickDmInput,
            onValueChange = onDmInputChange,
            modifier = Modifier.fillMaxWidth(),
            label = { Text("Find user or open DM") },
            placeholder = { Text("username") },
            singleLine = true,
            colors = pipeTextFieldColors(),
        )
        Spacer(modifier = Modifier.height(8.dp))
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            TextButton(onClick = onCreateDm) {
                Text("DM")
            }
            TextButton(onClick = { onLookupUser(state.quickDmInput) }) {
                Text("Profile")
            }
            TextButton(onClick = onOpenMembers) {
                Text("People")
            }
            TextButton(onClick = onOpenModLog) {
                Text("Log")
            }
        }
    }
}

@Composable
private fun SectionLabel(text: String) {
    Text(
        text = text,
        modifier = Modifier.padding(horizontal = 16.dp, vertical = 6.dp),
        color = Color(0xFF7C2D12),
        fontWeight = FontWeight.Bold,
    )
}

@Composable
private fun ChatCard(chat: ChatSummary, selected: Boolean, onClick: () -> Unit) {
    Column(
        modifier = Modifier
            .width(208.dp)
            .padding(start = 12.dp, end = 2.dp)
            .clip(RoundedCornerShape(24.dp))
            .background(if (selected) Color(0xFF1F2937) else Color.White)
            .clickable(onClick = onClick)
            .padding(14.dp),
    ) {
        Text(
            text = chat.title,
            color = if (selected) Color.White else Color(0xFF111827),
            fontWeight = FontWeight.Bold,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
        Spacer(modifier = Modifier.height(4.dp))
        Text(
            text = chat.subtitle.ifBlank { if (chat.isPrivate) "Private chat" else "Public chat" },
            color = if (selected) Color(0xFFD1D5DB) else Color(0xFF667085),
            style = MaterialTheme.typography.bodySmall,
            maxLines = 2,
            overflow = TextOverflow.Ellipsis,
        )
        Spacer(modifier = Modifier.height(10.dp))
        Row {
            if (chat.unreadCount > 0) {
                Badge(text = chat.unreadCount.toString(), background = Color(0xFFC2410C))
            }
            if (chat.mentionCount > 0) {
                Spacer(modifier = Modifier.width(if (chat.unreadCount > 0) 6.dp else 0.dp))
                Badge(text = "@${chat.mentionCount}", background = Color(0xFF7C2D12))
            }
        }
    }
}

@Composable
private fun MessageTimeline(
    messages: List<MessageItem>,
    selectedMessageId: String?,
    onSelectMessage: (String) -> Unit,
    modifier: Modifier = Modifier,
) {
    LazyColumn(
        modifier = modifier.fillMaxWidth(),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        items(messages, key = { it.id }) { message ->
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(24.dp))
                    .background(
                        when {
                            message.id == selectedMessageId -> Color(0xFFF3D9BC)
                            message.isMine -> Color(0xFFDCEAD9)
                            else -> Color.White
                        },
                    )
                    .clickable { onSelectMessage(message.id) }
                    .padding(14.dp),
            ) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                ) {
                    Text(
                        text = message.author,
                        fontWeight = FontWeight.Bold,
                        color = Color(0xFF101828),
                    )
                    Row {
                        if (message.isEdited) {
                            Text(
                                text = "edited",
                                color = Color(0xFF667085),
                                style = MaterialTheme.typography.labelSmall,
                            )
                            Spacer(modifier = Modifier.width(8.dp))
                        }
                        Text(
                            text = message.timeLabel,
                            color = Color(0xFF667085),
                            style = MaterialTheme.typography.labelSmall,
                        )
                    }
                }
                if (message.replyPreview.isNotBlank()) {
                    Spacer(modifier = Modifier.height(8.dp))
                    Text(
                        text = message.replyPreview,
                        color = Color(0xFF7C2D12),
                        style = MaterialTheme.typography.bodySmall,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
                Spacer(modifier = Modifier.height(8.dp))
                Text(
                    text = message.body,
                    color = Color(0xFF1D2939),
                    style = MaterialTheme.typography.bodyMedium,
                )
            }
        }
    }
}

@Composable
private fun SelectedMessageBar(
    selectedMessage: MessageItem,
    isEditing: Boolean,
    onReply: () -> Unit,
    onEdit: () -> Unit,
    onDelete: () -> Unit,
    onCancel: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(18.dp))
            .background(Color(0xFFEADBC8))
            .padding(10.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = if (isEditing) "Editing message" else "Selected message",
                color = Color(0xFF101828),
                fontWeight = FontWeight.SemiBold,
            )
            Text(
                text = selectedMessage.body,
                color = Color(0xFF475467),
                style = MaterialTheme.typography.bodySmall,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
        Spacer(modifier = Modifier.width(8.dp))
        Row {
            TextButton(onClick = onReply) {
                Text("Reply")
            }
            if (selectedMessage.isMine) {
                TextButton(onClick = onEdit) {
                    Text("Edit")
                }
                TextButton(onClick = onDelete) {
                    Text("Delete", color = Color(0xFFB42318))
                }
            }
            TextButton(onClick = onCancel) {
                Text("Close")
            }
        }
    }
}

@Composable
private fun ReplyBanner(
    replyTarget: MessageItem?,
    isEditing: Boolean,
    onCancel: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(18.dp))
            .background(Color(0xFFF7E6D3))
            .padding(10.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = if (isEditing) "Editing current message" else "Replying to ${replyTarget?.author ?: "message"}",
                color = Color(0xFF7C2D12),
                fontWeight = FontWeight.Bold,
            )
            if (replyTarget != null && !isEditing) {
                Text(
                    text = replyTarget.body,
                    color = Color(0xFF475467),
                    style = MaterialTheme.typography.bodySmall,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
        TextButton(onClick = onCancel) {
            Text("Cancel")
        }
    }
}

@Composable
private fun InfoSheet(
    state: PipeUiState,
    canManageMembers: Boolean,
    onMemberInputChange: (String) -> Unit,
    onInviteMember: () -> Unit,
    onRemoveMember: () -> Unit,
    onClose: () -> Unit,
) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 12.dp, vertical = 8.dp)
            .clip(RoundedCornerShape(24.dp))
            .background(Color(0xFFFCFAF7))
            .padding(14.dp),
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = when (state.activeSheet) {
                    SheetContent.Members -> "People"
                    SheetContent.UserLookup -> "Profile"
                    SheetContent.ModLog -> "Moderation log"
                    SheetContent.None -> ""
                },
                fontWeight = FontWeight.Bold,
                color = Color(0xFF101828),
            )
            TextButton(onClick = onClose) {
                Text("Close")
            }
        }
        HorizontalDivider()
        Spacer(modifier = Modifier.height(8.dp))
        when (state.activeSheet) {
            SheetContent.Members -> MemberSheet(
                members = state.members,
                canManageMembers = canManageMembers,
                memberActionInput = state.memberActionInput,
                onMemberInputChange = onMemberInputChange,
                onInviteMember = onInviteMember,
                onRemoveMember = onRemoveMember,
            )
            SheetContent.UserLookup -> UserCard(state.lookedUpUser)
            SheetContent.ModLog -> ModLogList(state.modLog)
            SheetContent.None -> Unit
        }
    }
}

@Composable
private fun MemberSheet(
    members: List<UserSummary>,
    canManageMembers: Boolean,
    memberActionInput: String,
    onMemberInputChange: (String) -> Unit,
    onInviteMember: () -> Unit,
    onRemoveMember: () -> Unit,
) {
    if (canManageMembers) {
        OutlinedTextField(
            value = memberActionInput,
            onValueChange = onMemberInputChange,
            modifier = Modifier.fillMaxWidth(),
            label = { Text("Invite or remove user") },
            placeholder = { Text("username") },
            singleLine = true,
            colors = pipeTextFieldColors(),
        )
        Spacer(modifier = Modifier.height(8.dp))
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            Button(onClick = onInviteMember, enabled = memberActionInput.isNotBlank()) {
                Text("Invite")
            }
            TextButton(onClick = onRemoveMember) {
                Text("Remove", color = Color(0xFFB42318))
            }
        }
        Spacer(modifier = Modifier.height(12.dp))
    }
    MemberList(members)
}

@Composable
private fun MemberList(members: List<UserSummary>) {
    if (members.isEmpty()) {
        Text("No members to show.", color = Color(0xFF667085))
        return
    }
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        members.forEach { member ->
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(18.dp))
                    .background(Color.White)
                    .padding(12.dp),
            ) {
                Text(member.username, fontWeight = FontWeight.Bold, color = Color(0xFF101828))
                Text(member.role, color = Color(0xFF7C2D12), style = MaterialTheme.typography.labelMedium)
                if (member.bio.isNotBlank()) {
                    Spacer(modifier = Modifier.height(4.dp))
                    Text(member.bio, color = Color(0xFF475467), style = MaterialTheme.typography.bodySmall)
                }
            }
        }
    }
}

@Composable
private fun UserCard(user: UserSummary?) {
    if (user == null) {
        Text("User not loaded.", color = Color(0xFF667085))
        return
    }
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(18.dp))
            .background(Color.White)
            .padding(14.dp),
    ) {
        Text(user.username, fontWeight = FontWeight.Black, color = Color(0xFF101828))
        Spacer(modifier = Modifier.height(4.dp))
        Text("role: ${user.role}", color = Color(0xFF7C2D12))
        Spacer(modifier = Modifier.height(6.dp))
        Text(user.bio.ifBlank { "No bio set." }, color = Color(0xFF475467))
    }
}

@Composable
private fun ModLogList(items: List<ModLogItem>) {
    if (items.isEmpty()) {
        Text("Moderation log is empty.", color = Color(0xFF667085))
        return
    }
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        items.forEach { item ->
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(18.dp))
                    .background(Color.White)
                    .padding(12.dp),
            ) {
                Text("${item.actor} -> ${item.target}", fontWeight = FontWeight.Bold, color = Color(0xFF101828))
                Text("${item.action} on ${item.channel}", color = Color(0xFF7C2D12), style = MaterialTheme.typography.labelMedium)
                Spacer(modifier = Modifier.height(4.dp))
                Text(item.details, color = Color(0xFF475467), style = MaterialTheme.typography.bodySmall)
                Spacer(modifier = Modifier.height(4.dp))
                Text(item.createdAt, color = Color(0xFF98A2B3), style = MaterialTheme.typography.labelSmall)
            }
        }
    }
}

@Composable
private fun Badge(text: String, background: Color) {
    Box(
        modifier = Modifier
            .clip(RoundedCornerShape(999.dp))
            .background(background)
            .padding(horizontal = 9.dp, vertical = 4.dp),
    ) {
        Text(
            text = text,
            color = Color.White,
            style = MaterialTheme.typography.labelSmall,
        )
    }
}

@Composable
private fun pipeTextFieldColors() = OutlinedTextFieldDefaults.colors(
    focusedTextColor = Color(0xFF101828),
    unfocusedTextColor = Color(0xFF101828),
    disabledTextColor = Color(0xFF98A2B3),
    focusedLabelColor = Color(0xFF7C2D12),
    unfocusedLabelColor = Color(0xFF475467),
    focusedPlaceholderColor = Color(0xFF98A2B3),
    unfocusedPlaceholderColor = Color(0xFF98A2B3),
    focusedBorderColor = Color(0xFF7C2D12),
    unfocusedBorderColor = Color(0xFFD0D5DD),
    cursorColor = Color(0xFF7C2D12),
    focusedContainerColor = Color.Transparent,
    unfocusedContainerColor = Color.Transparent,
)
