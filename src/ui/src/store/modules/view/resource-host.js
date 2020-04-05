const state = {
    activeDirectory: null,
    directoryList: []
}

const getters = {
    activeDirectory: state => state.activeDirectory || {},
    directoryList: state => state.directoryList
}

const mutations = {
    setActiveDirectory (state, active) {
        state.activeDirectory = active
    },
    setDirectoryList (state, list) {
        state.directoryList = list
    },
    addDirectory (state, directory) {
        state.directoryList.splice(1, 0, directory)
    },
    updateDirectory (state, directory) {
        const index = state.directoryList.findIndex(data => data.bk_module_id === directory.bk_module_id)
        if (index > -1) {
            state.directoryList.splice(index, 1, directory)
        }
    },
    deleteDirectory (state, id) {
        const index = state.directoryList.findIndex(target => target.bk_module_id === id)
        if (index > -1) {
            state.directoryList.splice(index, 1)
        }
    }
}

export default {
    namespaced: true,
    state,
    getters,
    mutations
}
