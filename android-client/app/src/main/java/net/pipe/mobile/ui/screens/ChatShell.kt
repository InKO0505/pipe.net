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
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
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
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.TextUnit
import androidx.compose.ui.unit.TextUnitType
import androidx.compose.ui.unit.dp
import net.pipe.mobile.data.ChatSummary
import net.pipe.mobile.data.MessageItem
import net.pipe.mobile.data.ModLogItem
import net.pipe.mobile.data.UserSummary
import net.pipe.mobile.ui.PipeUiState
import net.pipe.mobile.ui.PipeViewModel
import net.pipe.mobile.ui.SheetContent
import net.pipe.mobile.ui.theme.PipeColors
import net.pipe.mobile.ui.theme.PipeSpace

@Composable
fun ChatShell(viewModel: PipeViewModel) {
    val state by viewModel.uiState.collectAsState()

    Surface(
        modifier = Modifier.fillMaxSize(),
        color = PipeColors.Bg,
    ) {
        if (state.token == null) {
            ConnectScreen(
                state = state,
                onEndpointChange = viewModel::updateEndpoint,
                onUsernameChange = viewModel::updateUsername,
                onPinChange = viewModel::updatePin,
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
    onPinChange: (String) -> Unit,
    onConnect: () -> Unit,
) {
    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(PipeColors.BgDeep)
            .padding(24.dp),
        contentAlignment = Alignment.Center,
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .clip(RoundedCornerShape(PipeSpace.rXLarge))
                .background(PipeColors.Surface1)
                .padding(20.dp),
        ) {
            Text(
                text = "pipe.net",
                style = MaterialTheme.typography.headlineMedium,
                fontWeight = FontWeight.Bold,
                color = PipeColors.Text,
            )
            Spacer(modifier = Modifier.height(4.dp))
            Text(
                text = "terminal social network",
                color = PipeColors.TextMuted,
                style = MaterialTheme.typography.bodySmall,
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
            Spacer(modifier = Modifier.height(PipeSpace.md))
            OutlinedTextField(
                value = state.usernameInput,
                onValueChange = onUsernameChange,
                modifier = Modifier.fillMaxWidth(),
                label = { Text("Login") },
                placeholder = { Text("inko") },
                singleLine = true,
                colors = pipeTextFieldColors(),
            )
            Spacer(modifier = Modifier.height(PipeSpace.md))
            OutlinedTextField(
                value = state.pinInput,
                onValueChange = onPinChange,
                modifier = Modifier.fillMaxWidth(),
                label = { Text("PIN") },
                placeholder = { Text("Set with /setpin in SSH client") },
                singleLine = true,
                visualTransformation = PasswordVisualTransformation(),
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password),
                colors = pipeTextFieldColors(),
            )
            if (state.error != null) {
                Spacer(modifier = Modifier.height(PipeSpace.md))
                Text(
                    text = state.error,
                    color = PipeColors.Danger,
                    style = MaterialTheme.typography.bodyMedium,
                )
            }
            Spacer(modifier = Modifier.height(20.dp))
            Button(
                onClick = onConnect,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(PipeSpace.minTap),
                enabled = !state.loading,
                colors = ButtonDefaults.buttonColors(
                    containerColor = PipeColors.Accent,
                    contentColor = PipeColors.AccentInk,
                    disabledContainerColor = PipeColors.Surface3,
                    disabledContentColor = PipeColors.TextDim,
                ),
            ) {
                if (state.loading) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(18.dp),
                        strokeWidth = 2.dp,
                        color = PipeColors.AccentInk,
                    )
                } else {
                    Text("Connect", fontWeight = FontWeight.SemiBold)
                }
            }
            Spacer(modifier = Modifier.height(PipeSpace.sm))
            Text(
                text = "Emulator: http://10.0.2.2:8080  ·  Real device: http://<LAN-IP>:8080",
                color = PipeColors.TextDim,
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

        SectionLabel(text = if (state.showingMentions) "Inbox" else "Chats")

        LazyRow(
            modifier = Modifier
                .fillMaxWidth()
                .padding(vertical = 6.dp),
            horizontalArrangement = Arrangement.spacedBy(PipeSpace.sm),
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
                .padding(horizontal = PipeSpace.md),
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
                .background(PipeColors.Surface1)
                .padding(PipeSpace.md),
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
                Spacer(modifier = Modifier.height(PipeSpace.sm))
            }
            if (replyTarget != null || state.editingMessageId != null) {
                ReplyBanner(
                    replyTarget = replyTarget,
                    isEditing = state.editingMessageId != null,
                    onCancel = onCancelEdit,
                )
                Spacer(modifier = Modifier.height(PipeSpace.sm))
            }
            if (state.error != null) {
                Text(
                    text = state.error,
                    color = PipeColors.Danger,
                    style = MaterialTheme.typography.bodySmall,
                )
                Spacer(modifier = Modifier.height(PipeSpace.sm))
            }
            OutlinedTextField(
                value = state.composerText,
                onValueChange = onDraftChange,
                modifier = Modifier.fillMaxWidth(),
                label = { Text(if (state.editingMessageId != null) "Edit message" else "Message") },
                placeholder = { Text(if (state.editingMessageId != null) "Update the selected message" else "Write something") },
                shape = RoundedCornerShape(PipeSpace.rMedium),
                colors = pipeTextFieldColors(),
            )
            Spacer(modifier = Modifier.height(PipeSpace.sm))
            Button(
                onClick = onSend,
                enabled = !state.loading && state.composerText.isNotBlank() && state.selectedChatId != null,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(PipeSpace.minTap),
                colors = ButtonDefaults.buttonColors(
                    containerColor = PipeColors.Accent,
                    contentColor = PipeColors.AccentInk,
                    disabledContainerColor = PipeColors.Surface3,
                    disabledContentColor = PipeColors.TextDim,
                ),
            ) {
                if (state.loading) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(18.dp),
                        strokeWidth = 2.dp,
                        color = PipeColors.AccentInk,
                    )
                } else {
                    Text(
                        text = if (state.editingMessageId != null) "Save" else "↵  Send",
                        fontWeight = FontWeight.SemiBold,
                    )
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
            .background(PipeColors.Surface1)
            .padding(horizontal = PipeSpace.lg, vertical = PipeSpace.md),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = "pipe.net",
                color = PipeColors.Text,
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Bold,
            )
            if (state.profile?.endpoint != null) {
                Text(
                    text = state.profile.endpoint,
                    color = PipeColors.TextDim,
                    style = MaterialTheme.typography.bodySmall,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
        Spacer(modifier = Modifier.width(PipeSpace.md))
        TextButton(onClick = onShowMentions) {
            Text("Inbox", color = PipeColors.TextSecondary)
        }
        TextButton(onClick = onRefresh) {
            Text("Sync", color = PipeColors.TextSecondary)
        }
    }
}

@Composable
private fun ProfileStrip(state: PipeUiState) {
    val profile = state.profile ?: return
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = PipeSpace.md, vertical = PipeSpace.sm)
            .clip(RoundedCornerShape(PipeSpace.rLarge))
            .background(PipeColors.Surface2)
            .padding(PipeSpace.md),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            modifier = Modifier
                .size(40.dp)
                .clip(RoundedCornerShape(PipeSpace.rMedium))
                .background(PipeColors.Surface3),
            contentAlignment = Alignment.Center,
        ) {
            val glyph = when (profile.role) {
                "owner" -> "👑"
                "admin" -> "★"
                else -> profile.username.take(1).uppercase()
            }
            Text(
                text = glyph,
                color = if (profile.role == "owner") PipeColors.Accent else PipeColors.Text,
                fontWeight = FontWeight.Bold,
            )
        }
        Spacer(modifier = Modifier.width(PipeSpace.md))
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = profile.role.uppercase(),
                color = PipeColors.Accent,
                style = MaterialTheme.typography.labelSmall,
                fontWeight = FontWeight.Bold,
                letterSpacing = TextUnit(0.18f, TextUnitType.Em),
            )
            Text(
                text = profile.username,
                color = PipeColors.Text,
                style = MaterialTheme.typography.bodyMedium,
                fontWeight = FontWeight.SemiBold,
            )
            if (profile.bio.isNotBlank()) {
                Text(
                    text = profile.bio,
                    color = PipeColors.TextMuted,
                    style = MaterialTheme.typography.bodySmall,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
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
            .padding(horizontal = PipeSpace.md)
            .clip(RoundedCornerShape(PipeSpace.rLarge))
            .background(PipeColors.Surface1)
            .padding(PipeSpace.md),
    ) {
        Text(
            text = "NEW CHANNEL",
            color = PipeColors.TextMuted,
            style = MaterialTheme.typography.labelSmall,
            fontWeight = FontWeight.Bold,
            letterSpacing = TextUnit(0.18f, TextUnitType.Em),
        )
        Spacer(modifier = Modifier.height(PipeSpace.sm))
        OutlinedTextField(
            value = state.newChannelName,
            onValueChange = onNameChange,
            modifier = Modifier.fillMaxWidth(),
            label = { Text("Channel name") },
            singleLine = true,
            colors = pipeTextFieldColors(),
        )
        Spacer(modifier = Modifier.height(PipeSpace.sm))
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = if (state.newChannelPrivate) "Private" else "Public",
                color = PipeColors.TextSecondary,
                style = MaterialTheme.typography.bodySmall,
            )
            Row {
                TextButton(onClick = onTogglePrivacy) {
                    Text(
                        text = if (state.newChannelPrivate) "Make public" else "Make private",
                        color = PipeColors.TextSecondary,
                    )
                }
                Button(
                    onClick = onCreate,
                    enabled = state.newChannelName.isNotBlank() && !state.loading,
                    colors = ButtonDefaults.buttonColors(
                        containerColor = PipeColors.Accent,
                        contentColor = PipeColors.AccentInk,
                        disabledContainerColor = PipeColors.Surface3,
                        disabledContentColor = PipeColors.TextDim,
                    ),
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
            .padding(horizontal = PipeSpace.md)
            .clip(RoundedCornerShape(PipeSpace.rLarge))
            .background(PipeColors.Surface1)
            .padding(PipeSpace.md),
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
        Spacer(modifier = Modifier.height(PipeSpace.xs))
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            TextButton(onClick = onCreateDm) { Text("DM", color = PipeColors.TextSecondary) }
            TextButton(onClick = { onLookupUser(state.quickDmInput) }) { Text("Profile", color = PipeColors.TextSecondary) }
            TextButton(onClick = onOpenMembers) { Text("People", color = PipeColors.TextSecondary) }
            TextButton(onClick = onOpenModLog) { Text("Log", color = PipeColors.TextSecondary) }
        }
    }
}

@Composable
private fun SectionLabel(text: String) {
    Text(
        text = text.uppercase(),
        modifier = Modifier.padding(horizontal = PipeSpace.lg, vertical = PipeSpace.xs),
        color = PipeColors.TextMuted,
        style = MaterialTheme.typography.labelSmall,
        fontWeight = FontWeight.Bold,
        letterSpacing = TextUnit(0.18f, TextUnitType.Em),
    )
}

@Composable
private fun ChatCard(chat: ChatSummary, selected: Boolean, onClick: () -> Unit) {
    Column(
        modifier = Modifier
            .width(200.dp)
            .padding(start = PipeSpace.md, end = PipeSpace.xs)
            .clip(RoundedCornerShape(PipeSpace.rLarge))
            .background(if (selected) PipeColors.Surface2 else PipeColors.Surface1)
            .clickable(onClick = onClick)
            .padding(PipeSpace.md),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            if (selected) {
                Box(
                    modifier = Modifier
                        .width(2.dp)
                        .height(14.dp)
                        .background(PipeColors.Accent),
                )
                Spacer(modifier = Modifier.width(PipeSpace.xs))
            }
            Text(
                text = chat.title,
                color = if (selected) PipeColors.Text else PipeColors.TextSecondary,
                fontWeight = if (selected) FontWeight.SemiBold else FontWeight.Normal,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                style = MaterialTheme.typography.bodyMedium,
            )
        }
        Spacer(modifier = Modifier.height(PipeSpace.xs))
        Text(
            text = chat.subtitle.ifBlank { if (chat.isPrivate) "⊡ private" else "# public" },
            color = PipeColors.TextDim,
            style = MaterialTheme.typography.bodySmall,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
        if (chat.unreadCount > 0 || chat.mentionCount > 0) {
            Spacer(modifier = Modifier.height(PipeSpace.xs))
            Row {
                if (chat.unreadCount > 0) {
                    Badge(text = chat.unreadCount.toString(), background = PipeColors.Accent, textColor = PipeColors.AccentInk)
                }
                if (chat.mentionCount > 0) {
                    Spacer(modifier = Modifier.width(if (chat.unreadCount > 0) PipeSpace.xs else 0.dp))
                    Badge(text = "@${chat.mentionCount}", background = PipeColors.Accent, textColor = PipeColors.AccentInk)
                }
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
        verticalArrangement = Arrangement.spacedBy(PipeSpace.sm),
    ) {
        items(messages, key = { it.id }) { message ->
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(PipeSpace.rLarge))
                    .background(
                        when {
                            message.id == selectedMessageId -> PipeColors.Surface3
                            message.isMine -> PipeColors.Surface2
                            else -> PipeColors.Surface1
                        },
                    )
                    .clickable { onSelectMessage(message.id) }
                    .padding(PipeSpace.md),
            ) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                ) {
                    Text(
                        text = message.author,
                        fontWeight = FontWeight.SemiBold,
                        color = PipeColors.Text,
                        style = MaterialTheme.typography.bodyMedium,
                    )
                    Row {
                        if (message.isEdited) {
                            Text(
                                text = "edited",
                                color = PipeColors.TextDim,
                                style = MaterialTheme.typography.labelSmall,
                            )
                            Spacer(modifier = Modifier.width(PipeSpace.sm))
                        }
                        Text(
                            text = message.timeLabel,
                            color = PipeColors.TextDim,
                            style = MaterialTheme.typography.labelSmall,
                        )
                    }
                }
                if (message.replyPreview.isNotBlank()) {
                    Spacer(modifier = Modifier.height(PipeSpace.xs))
                    Text(
                        text = "↳ ${message.replyPreview}",
                        color = PipeColors.TextMuted,
                        style = MaterialTheme.typography.bodySmall,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
                Spacer(modifier = Modifier.height(PipeSpace.xs))
                Text(
                    text = message.body,
                    color = PipeColors.TextSecondary,
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
            .clip(RoundedCornerShape(PipeSpace.rLarge))
            .background(PipeColors.Surface2)
            .padding(PipeSpace.sm),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = if (isEditing) "Editing" else "Selected",
                color = PipeColors.TextMuted,
                style = MaterialTheme.typography.labelSmall,
                fontWeight = FontWeight.SemiBold,
            )
            Text(
                text = selectedMessage.body,
                color = PipeColors.TextSecondary,
                style = MaterialTheme.typography.bodySmall,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
        Spacer(modifier = Modifier.width(PipeSpace.sm))
        Row {
            TextButton(onClick = onReply) { Text("Reply", color = PipeColors.TextSecondary) }
            if (selectedMessage.isMine) {
                TextButton(onClick = onEdit) { Text("Edit", color = PipeColors.TextSecondary) }
                TextButton(onClick = onDelete) { Text("Delete", color = PipeColors.Danger) }
            }
            TextButton(onClick = onCancel) { Text("✕", color = PipeColors.TextMuted) }
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
            .clip(RoundedCornerShape(PipeSpace.rLarge))
            .background(PipeColors.Surface2)
            .padding(PipeSpace.sm),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = if (isEditing) "Editing message" else "↳ reply to ${replyTarget?.author ?: "message"}",
                color = PipeColors.Accent,
                fontWeight = FontWeight.SemiBold,
                style = MaterialTheme.typography.bodySmall,
            )
            if (replyTarget != null && !isEditing) {
                Text(
                    text = replyTarget.body,
                    color = PipeColors.TextMuted,
                    style = MaterialTheme.typography.bodySmall,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
        TextButton(onClick = onCancel) { Text("✕", color = PipeColors.TextMuted) }
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
            .padding(horizontal = PipeSpace.md, vertical = PipeSpace.sm)
            .clip(RoundedCornerShape(PipeSpace.rLarge))
            .background(PipeColors.Surface1)
            .padding(PipeSpace.md),
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = when (state.activeSheet) {
                    SheetContent.Members -> "PEOPLE"
                    SheetContent.UserLookup -> "PROFILE"
                    SheetContent.ModLog -> "MOD LOG"
                    SheetContent.None -> ""
                },
                fontWeight = FontWeight.Bold,
                color = PipeColors.TextMuted,
                style = MaterialTheme.typography.labelSmall,
                letterSpacing = TextUnit(0.18f, TextUnitType.Em),
            )
            TextButton(onClick = onClose) { Text("✕", color = PipeColors.TextMuted) }
        }
        HorizontalDivider(color = PipeColors.BorderSubtle)
        Spacer(modifier = Modifier.height(PipeSpace.sm))
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
        Spacer(modifier = Modifier.height(PipeSpace.sm))
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            Button(
                onClick = onInviteMember,
                enabled = memberActionInput.isNotBlank(),
                colors = ButtonDefaults.buttonColors(
                    containerColor = PipeColors.Accent,
                    contentColor = PipeColors.AccentInk,
                ),
            ) { Text("Invite") }
            TextButton(onClick = onRemoveMember) {
                Text("Remove", color = PipeColors.Danger)
            }
        }
        Spacer(modifier = Modifier.height(PipeSpace.md))
    }
    MemberList(members)
}

@Composable
private fun MemberList(members: List<UserSummary>) {
    if (members.isEmpty()) {
        Text("No members.", color = PipeColors.TextMuted, style = MaterialTheme.typography.bodySmall)
        return
    }
    Column(verticalArrangement = Arrangement.spacedBy(PipeSpace.sm)) {
        members.forEach { member ->
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(PipeSpace.rMedium))
                    .background(PipeColors.Surface2)
                    .padding(PipeSpace.md),
            ) {
                Text(member.username, fontWeight = FontWeight.SemiBold, color = PipeColors.Text)
                Text(
                    text = member.role.uppercase(),
                    color = when (member.role) {
                        "owner" -> PipeColors.Accent
                        "admin" -> PipeColors.Text
                        else -> PipeColors.TextMuted
                    },
                    style = MaterialTheme.typography.labelSmall,
                    letterSpacing = TextUnit(0.18f, TextUnitType.Em),
                )
                if (member.bio.isNotBlank()) {
                    Spacer(modifier = Modifier.height(PipeSpace.xs))
                    Text(member.bio, color = PipeColors.TextMuted, style = MaterialTheme.typography.bodySmall)
                }
            }
        }
    }
}

@Composable
private fun UserCard(user: UserSummary?) {
    if (user == null) {
        Text("User not found.", color = PipeColors.TextMuted, style = MaterialTheme.typography.bodySmall)
        return
    }
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(PipeSpace.rMedium))
            .background(PipeColors.Surface2)
            .padding(PipeSpace.md),
    ) {
        Text(user.username, fontWeight = FontWeight.Bold, color = PipeColors.Text)
        Spacer(modifier = Modifier.height(PipeSpace.xs))
        Text(
            text = user.role.uppercase(),
            color = when (user.role) {
                "owner" -> PipeColors.Accent
                "admin" -> PipeColors.Text
                else -> PipeColors.TextMuted
            },
            style = MaterialTheme.typography.labelSmall,
            letterSpacing = TextUnit(0.18f, TextUnitType.Em),
        )
        if (user.bio.isNotBlank()) {
            Spacer(modifier = Modifier.height(PipeSpace.xs))
            Text(user.bio, color = PipeColors.TextMuted, style = MaterialTheme.typography.bodySmall)
        }
    }
}

@Composable
private fun ModLogList(items: List<ModLogItem>) {
    if (items.isEmpty()) {
        Text("Log is empty.", color = PipeColors.TextMuted, style = MaterialTheme.typography.bodySmall)
        return
    }
    Column(verticalArrangement = Arrangement.spacedBy(PipeSpace.sm)) {
        items.forEach { item ->
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(PipeSpace.rMedium))
                    .background(PipeColors.Surface2)
                    .padding(PipeSpace.md),
            ) {
                Text("${item.actor} → ${item.target}", fontWeight = FontWeight.SemiBold, color = PipeColors.Text)
                Text(
                    text = "${item.action} · ${item.channel}",
                    color = PipeColors.Accent,
                    style = MaterialTheme.typography.labelSmall,
                    letterSpacing = TextUnit(0.18f, TextUnitType.Em),
                )
                Spacer(modifier = Modifier.height(PipeSpace.xs))
                Text(item.details, color = PipeColors.TextSecondary, style = MaterialTheme.typography.bodySmall)
                Spacer(modifier = Modifier.height(PipeSpace.xs))
                Text(item.createdAt, color = PipeColors.TextDim, style = MaterialTheme.typography.labelSmall)
            }
        }
    }
}

@Composable
private fun Badge(text: String, background: Color, textColor: Color = PipeColors.AccentInk) {
    Box(
        modifier = Modifier
            .clip(RoundedCornerShape(999.dp))
            .background(background)
            .padding(horizontal = PipeSpace.sm, vertical = PipeSpace.xs),
    ) {
        Text(
            text = text,
            color = textColor,
            style = MaterialTheme.typography.labelSmall,
            fontWeight = FontWeight.Bold,
        )
    }
}

@Composable
private fun pipeTextFieldColors() = OutlinedTextFieldDefaults.colors(
    focusedTextColor = PipeColors.Text,
    unfocusedTextColor = PipeColors.TextSecondary,
    disabledTextColor = PipeColors.TextDim,
    focusedLabelColor = PipeColors.Accent,
    unfocusedLabelColor = PipeColors.TextMuted,
    focusedPlaceholderColor = PipeColors.TextDim,
    unfocusedPlaceholderColor = PipeColors.TextDim,
    focusedBorderColor = PipeColors.Accent,
    unfocusedBorderColor = PipeColors.Border,
    cursorColor = PipeColors.Accent,
    focusedContainerColor = Color.Transparent,
    unfocusedContainerColor = Color.Transparent,
)
