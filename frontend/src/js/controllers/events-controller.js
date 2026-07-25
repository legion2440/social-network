(function (root, factory) {
  var create = factory(root && root.createFeatureController, root && root.createControllerContext);
  if (typeof module === 'object' && module.exports) module.exports = create;
  if (root) root.createEventsController = create;
})(typeof window !== 'undefined' ? window : globalThis, function (createController, createContext) {
  return function createEventsController(dependencies) {
    var context = createContext(dependencies);
    var AuthAPI = dependencies.api;
    var UserModel = dependencies.models.users;
    var GroupEventModel = dependencies.models.events;
    var requestErrorMessage = dependencies.helpers.requestErrorMessage;
    var parseLocalDateTime = dependencies.helpers.parseLocalDateTime;
    var formatDateTimeInput = dependencies.helpers.formatDateTimeInput;
    var num = dependencies.helpers.num;

  context.groupEventResponseGate = function (eventID) {
    const key = String(Number(eventID));
    if (!context.groupEventResponseGatesByID[key]) {
      context.groupEventResponseGatesByID[key] = UserModel.createRequestGate();
    }
    return context.groupEventResponseGatesByID[key];
  };

  context.invalidateGroupEventResponses = function () {
    Object.keys(context.groupEventResponseGatesByID).forEach(key => {
      context.groupEventResponseGatesByID[key].begin();
    });
    context.groupEventResponseGatesByID = {};
  };

  context.loadGroupEvents = async (groupID, reset = true) => {
    groupID = Number(groupID);
    if (!Number.isInteger(groupID) || groupID <= 0 || context.groupAccessIsRevoked(groupID)) return;
    const group = context.state.apiGroupsByID[String(groupID)];
    if (group && group.state !== 'owner' && group.state !== 'member') return;
    const authGeneration = context.authGate.current();
    const accessGate = context.groupGeneration(groupID);
    const accessGeneration = accessGate.current();
    const generation = reset ? context.groupEventsGate.begin() : context.groupEventsGate.current();
    if (!reset && context.state.groupEventsPending) return;
    const cursor = reset ? null : context.state.groupEventsNextCursor;
    if (!reset && !cursor) return;
    context.setState({ groupEventsPending: true, groupEventsLoading: !!reset, groupEventsError: '' });
    try {
      const page = await AuthAPI.groupEvents(groupID, cursor, 20);
      if (
        !context.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
        !context.groupEventsGate.isCurrent(generation) || context.groupAccessIsRevoked(groupID) ||
        Number(context.state.groupId) !== groupID
      ) return;
      const rawEvents = page.events || [];
      const mapped = rawEvents.map(event => GroupEventModel.normalizeEventResponse(event));
      const apiUsersByID = context.mergeAPIUsers(rawEvents.map(event => event.creator));
      context.setState(current => {
        let events = reset ? [] : current.groupEvents.slice();
        mapped.forEach(event => { events = GroupEventModel.mergeAuthoritative(events, event); });
        return {
          apiUsersByID, groupEvents: events,
          groupEventsNextCursor: page.next_cursor || null,
          groupEventsPending: false, groupEventsLoading: false, groupEventsError: ''
        };
      });
    } catch (error) {
      if (
        !context.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
        !context.groupEventsGate.isCurrent(generation) || context.groupAccessIsRevoked(groupID) ||
        Number(context.state.groupId) !== groupID
      ) return;
      if (error && error.status === 403) {
        context.revokeGroupAccess(groupID);
        return;
      }
      context.setState({
        groupEventsPending: false, groupEventsLoading: false,
        groupEventsError: requestErrorMessage(error, error && error.status === 404 ? 'Group not found.' : 'Could not load group events.')
      });
    }
  };

  context.createGroupEvent = async () => {
    const groupID = Number(context.state.groupId);
    const group = context.state.apiGroupsByID[String(groupID)];
    const startsAt = parseLocalDateTime(context.state.groupEventStartsAt);
    if (
      !Number.isInteger(groupID) || groupID <= 0 || context.groupAccessIsRevoked(groupID) ||
      !group || (group.state !== 'owner' && group.state !== 'member') ||
      context.state.groupEventCreatePending || !context.state.groupEventTitle.trim() ||
      !context.state.groupEventDescription.trim() || !startsAt
    ) return;
    const authGeneration = context.authGate.current();
    const accessGate = context.groupGeneration(groupID);
    const accessGeneration = accessGate.current();
    const createGeneration = context.groupEventCreateGate.current();
    context.setState({ groupEventCreatePending: true, groupEventCreateError: '' });
    try {
      const raw = await AuthAPI.createGroupEvent(groupID, {
        title: context.state.groupEventTitle.trim(),
        description: context.state.groupEventDescription.trim(),
        starts_at: startsAt.toISOString()
      });
      if (
        !context.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
        !context.groupEventCreateGate.isCurrent(createGeneration) || context.groupAccessIsRevoked(groupID) ||
        Number(context.state.groupId) !== groupID
      ) return;
      context.groupEventsGate.begin();
      const event = GroupEventModel.normalizeEventResponse(raw);
      const apiUsersByID = context.mergeAPIUsers([raw.creator]);
      context.setState(current => ({
        apiUsersByID,
        groupEvents: GroupEventModel.mergeAuthoritative(current.groupEvents, event),
        groupEventsPending: false, groupEventsLoading: false,
        groupEventComposerOpen: false, groupEventTitle: '', groupEventDescription: '', groupEventStartsAt: '',
        groupEventCreatePending: false, groupEventCreateError: ''
      }));
    } catch (error) {
      if (
        !context.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
        !context.groupEventCreateGate.isCurrent(createGeneration) || context.groupAccessIsRevoked(groupID) ||
        Number(context.state.groupId) !== groupID
      ) return;
      if (error && error.status === 403) {
        context.revokeGroupAccess(groupID);
        return;
      }
      context.setState({
        groupEventCreatePending: false,
        groupEventCreateError: requestErrorMessage(error, 'Could not create the event. Your draft was kept.')
      });
    }
  };

  context.respondToGroupEvent = async (eventID, response) => {
    const groupID = Number(context.state.groupId);
    eventID = Number(eventID);
    const key = String(eventID);
    if (
      !Number.isInteger(groupID) || groupID <= 0 || !Number.isInteger(eventID) || eventID <= 0 ||
      (response !== 'going' && response !== 'not_going') || context.groupAccessIsRevoked(groupID) ||
      context.state.groupEventResponsePendingByID[key]
    ) return;
    const authGeneration = context.authGate.current();
    const accessGate = context.groupGeneration(groupID);
    const accessGeneration = accessGate.current();
    const responseGate = context.groupEventResponseGate(eventID);
    const responseGeneration = responseGate.current();
    context.setState({
      groupEventResponsePendingByID: Object.assign({}, context.state.groupEventResponsePendingByID, { [key]: true }),
      groupEventResponseErrorByID: Object.assign({}, context.state.groupEventResponseErrorByID, { [key]: '' })
    });
    try {
      const raw = await AuthAPI.respondToGroupEvent(groupID, eventID, response);
      if (
        !context.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
        !responseGate.isCurrent(responseGeneration) || context.groupAccessIsRevoked(groupID) ||
        Number(context.state.groupId) !== groupID
      ) return;
      context.groupEventsGate.begin();
      const event = GroupEventModel.normalizeEventResponse(raw);
      const apiUsersByID = context.mergeAPIUsers([raw.creator]);
      context.setState(current => ({
        apiUsersByID,
        groupEvents: GroupEventModel.mergeAuthoritative(current.groupEvents, event),
        groupEventsPending: false, groupEventsLoading: false,
        groupEventResponsePendingByID: Object.assign({}, current.groupEventResponsePendingByID, { [key]: false }),
        groupEventResponseErrorByID: Object.assign({}, current.groupEventResponseErrorByID, { [key]: '' })
      }));
    } catch (error) {
      if (
        !context.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
        !responseGate.isCurrent(responseGeneration) || context.groupAccessIsRevoked(groupID) ||
        Number(context.state.groupId) !== groupID
      ) return;
      if (error && error.status === 403) {
        context.revokeGroupAccess(groupID);
        return;
      }
      context.setState(current => ({
        groupEventResponsePendingByID: Object.assign({}, current.groupEventResponsePendingByID, { [key]: false }),
        groupEventResponseErrorByID: Object.assign({}, current.groupEventResponseErrorByID, {
          [key]: requestErrorMessage(error, 'Could not update your response.')
        })
      }));
    }
  };

  context.toggleComposer = () => context.setState({
    groupEventComposerOpen: !context.state.groupEventComposerOpen,
    groupEventCreateError: ''
  });

  context.updateComposerField = (field, value) => {
    const stateField = {
      title: 'groupEventTitle',
      description: 'groupEventDescription',
      startsAt: 'groupEventStartsAt'
    }[field];
    if (!stateField) return;
    context.setState({
      [stateField]: field === 'startsAt' ? formatDateTimeInput(value) : value,
      groupEventCreateError: ''
    });
  };

    return createController('events', dependencies, {
      groupEventResponseGate: context.groupEventResponseGate,
      invalidateGroupEventResponses: context.invalidateGroupEventResponses,
      loadGroupEvents: context.loadGroupEvents,
      createGroupEvent: context.createGroupEvent,
      respondToGroupEvent: context.respondToGroupEvent,
      toggleComposer: context.toggleComposer,
      updateComposerField: context.updateComposerField
    }, function (state) {
      var groupID = Number(state.groupId);
      var events = state.groupEvents.map(function (event, index) {
        var eventID = String(event.id);
        var startsAt = new Date(event.startsAt);
        var pending = !!state.groupEventResponsePendingByID[eventID];
        return {
          id: event.id,
          title: event.title,
          description: event.description,
          startsAt: Number.isNaN(startsAt.getTime())
            ? event.startsAt
            : startsAt.toLocaleString([], {
              dateStyle: 'medium',
              timeStyle: 'short'
            }),
          creator: context.apiUser(event.creatorID),
          goingCount: num(event.goingCount),
          notGoingCount: num(event.notGoingCount),
          goingSelected: event.viewerResponse === 'going',
          notGoingSelected: event.viewerResponse === 'not_going',
          goingBg: event.viewerResponse === 'going'
            ? 'var(--accent)'
            : 'transparent',
          goingColor: event.viewerResponse === 'going'
            ? '#fff'
            : 'var(--text2)',
          notGoingBg: event.viewerResponse === 'not_going'
            ? 'var(--surface2)'
            : 'transparent',
          notGoingColor: event.viewerResponse === 'not_going'
            ? 'var(--text)'
            : 'var(--text2)',
          pending: pending,
          error: state.groupEventResponseErrorByID[eventID] || '',
          hasError: !!state.groupEventResponseErrorByID[eventID],
          delay: (index * 0.05).toFixed(2) + 's',
          goProfile: function () { return context.openProfile(event.creatorID); },
          going: function () {
            return context.respondToGroupEvent(event.id, 'going');
          },
          notGoing: function () {
            return context.respondToGroupEvent(event.id, 'not_going');
          }
        };
      });
      var startsAt = parseLocalDateTime(state.groupEventStartsAt);
      var createDisabled = state.groupEventCreatePending ||
        !state.groupEventTitle.trim() ||
        !state.groupEventDescription.trim() ||
        !startsAt;

      return {
        gEvents: events,
        groupEventsLoading: state.groupEventsLoading,
        groupEventsHasError: !!state.groupEventsError,
        groupEventsError: state.groupEventsError,
        groupEventsEmpty: !state.groupEventsLoading &&
          !state.groupEventsError && events.length === 0,
        groupEventsHasMore: !!state.groupEventsNextCursor,
        groupEventsLoadMoreDisabled: state.groupEventsPending,
        retryGroupEvents: function () {
          return context.loadGroupEvents(groupID, true);
        },
        loadMoreGroupEvents: function () {
          return context.loadGroupEvents(groupID, false);
        },
        groupEventComposerOpen: state.groupEventComposerOpen,
        toggleGroupEventComposer: context.toggleComposer,
        groupEventTitle: state.groupEventTitle,
        onGroupEventTitle: function (event) {
          return context.updateComposerField('title', event.target.value);
        },
        groupEventDescription: state.groupEventDescription,
        onGroupEventDescription: function (event) {
          return context.updateComposerField('description', event.target.value);
        },
        groupEventStartsAt: state.groupEventStartsAt,
        onGroupEventStartsAt: function (event) {
          return context.updateComposerField('startsAt', event.target.value);
        },
        groupEventCreatePending: state.groupEventCreatePending,
        groupEventCreateHasError: !!state.groupEventCreateError,
        groupEventCreateError: state.groupEventCreateError,
        groupEventCreateDisabled: createDisabled,
        groupEventCreateButtonLabel: state.groupEventCreatePending
          ? 'Creating…'
          : 'Create event',
        createGroupEvent: context.createGroupEvent,
        railEvents: []
      };
    }, {});
  };
});
